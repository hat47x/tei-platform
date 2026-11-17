//go:build linux && amd64

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	ptraceEventExec = 4

	ptraceOTraceSysgood = 0x1
	ptraceOTraceFork    = 0x2
	ptraceOTraceVfork   = 0x4
	ptraceOTraceClone   = 0x8
	ptraceOTraceExec    = 0x10
	ptraceOExitKill     = 0x100000

	atFdcwd = -100
)

type fileKey struct {
	path, image string
	step        int
}

type tracee struct {
	image   string
	execSeq int
	step    int
	pending *pendingCall
}

// observation is what one run has gathered so far.
type observation struct {
	uses   map[fileKey]*FileUse
	limits Limits
}

func observe(args []string) int {
	flags := flag.NewFlagSet("observe", flag.ContinueOnError)
	out := flags.String("out", "", "file to write the observation record to")
	files := flags.Bool("files", false, "also record opened paths (slower: stops on every system call)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *out == "" || flags.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: reference-build-process-observation observe -out record.json [-files] -- command [args...]")
		return 2
	}
	code, err := run(*out, *files, flags.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "observation failed:", err)
		return 125
	}
	return code
}

func run(out string, files bool, command []string) (int, error) {
	// Every ptrace request for a tracee must come from the thread that attached.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Ptrace: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	root := cmd.Process.Pid
	var ws syscall.WaitStatus
	if _, err := syscall.Wait4(root, &ws, 0, nil); err != nil {
		return 0, err
	}
	options := ptraceOTraceFork | ptraceOTraceVfork | ptraceOTraceClone | ptraceOTraceExec | ptraceOExitKill
	if files {
		options |= ptraceOTraceSysgood
	}
	if err := syscall.PtraceSetOptions(root, options); err != nil {
		return 0, err
	}
	resume := func(pid, sig int) {
		if files {
			_ = syscall.PtraceSyscall(pid, sig)
		} else {
			_ = syscall.PtraceCont(pid, sig)
		}
	}

	record := Record{
		Schema:      observationSchema,
		Command:     command,
		FilesTraced: files,
		NotObserved: []string{
			"what a process did with an opened descriptor (bytes read, bytes written)",
			"reads through descriptors inherited from outside the traced tree",
			"network traffic, and anything a process learned from the clock or from random numbers",
			"values of environment variables (only their names are recorded)",
			"processes outside the traced tree that the build talked to (daemons, servers)",
		},
	}
	if !files {
		record.NotObserved = append(record.NotObserved, "opened paths (run with -files)")
	}
	// The first image is the command itself; its exec stop was consumed above.
	first := describeExec(0, root)
	first.ParentSeq = -1
	record.Execs = append(record.Execs, first)

	tracees := map[int]*tracee{root: {}}
	seen := &observation{uses: map[fileKey]*FileUse{}}
	exitCode := 0
	resume(root, 0)
	for len(tracees) > 0 {
		pid, err := syscall.Wait4(-1, &ws, syscall.WALL, nil)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if err == syscall.ECHILD {
				break
			}
			return 0, err
		}
		if ws.Exited() || ws.Signaled() {
			delete(tracees, pid)
			if pid == root {
				if ws.Exited() {
					exitCode = ws.ExitStatus()
				} else {
					exitCode = 128 + int(ws.Signal())
				}
			}
			continue
		}
		if !ws.Stopped() {
			continue
		}
		state, known := tracees[pid]
		if !known {
			state = &tracee{}
			if origin := originOf(pid, tracees); origin != nil {
				state.execSeq, state.step = origin.execSeq, origin.step
			} else {
				// Its parent is gone or unknown. Crediting it to the command
				// itself would let it read what only the command may read.
				seen.limits.UnattributedTasks++
			}
			tracees[pid] = state
		}
		sig := ws.StopSignal()
		switch {
		case sig == syscall.SIGTRAP|0x80:
			seen.syscallStop(pid, state)
			resume(pid, 0)
		case sig == syscall.SIGTRAP && ws.TrapCause() == ptraceEventExec:
			state.image = ""
			e := describeExec(len(record.Execs), pid)
			e.ParentSeq = state.execSeq
			if state.execSeq == 0 {
				state.step = e.Seq
			}
			state.execSeq = e.Seq
			e.Step = state.step
			record.Execs = append(record.Execs, e)
			resume(pid, 0)
		case sig == syscall.SIGTRAP && ws.TrapCause() > 0:
			resume(pid, 0)
		case sig == syscall.SIGSTOP && !known:
			// The first stop of an automatically attached child.
			resume(pid, 0)
		default:
			resume(pid, int(sig))
		}
	}

	if files {
		for _, use := range seen.uses {
			record.Files = append(record.Files, *use)
		}
		sort.Slice(record.Files, func(i, j int) bool {
			a, b := record.Files[i], record.Files[j]
			if a.Path != b.Path {
				return a.Path < b.Path
			}
			if a.Step != b.Step {
				return a.Step < b.Step
			}
			return a.Image < b.Image
		})
	}
	record.Limits = seen.limits
	record.ExitCode = exitCode
	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(out, append(encoded, '\n'), 0o644); err != nil {
		return 0, err
	}
	return exitCode, nil
}

func describeExec(seq, pid int) Exec {
	proc := "/proc/" + strconv.Itoa(pid)
	e := Exec{Seq: seq, PID: pid}
	e.Image, _ = os.Readlink(proc + "/exe")
	e.ImageDigest = imageDigest(proc + "/exe")
	e.Cwd, _ = os.Readlink(proc + "/cwd")
	if raw, err := os.ReadFile(proc + "/cmdline"); err == nil {
		e.Argv = splitNul(raw)
	}
	if raw, err := os.ReadFile(proc + "/environ"); err == nil {
		for _, entry := range splitNul(raw) {
			name, _, _ := strings.Cut(entry, "=")
			e.EnvNames = append(e.EnvNames, name)
		}
		sort.Strings(e.EnvNames)
	}
	if raw, err := os.ReadFile(proc + "/stat"); err == nil {
		// The command name may contain spaces; the fields after the last ')' are fixed.
		if i := strings.LastIndexByte(string(raw), ')'); i >= 0 {
			fields := strings.Fields(string(raw[i+1:]))
			if len(fields) > 1 {
				e.ParentPID, _ = strconv.Atoi(fields[1])
			}
		}
	}
	return e
}

func splitNul(raw []byte) []string {
	text := strings.TrimRight(string(raw), "\x00")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\x00")
}

// syscallInfo mirrors the head of the kernel's struct ptrace_syscall_info.
// The kernel says whether a stop is an entry or an exit; guessing it by
// counting stops goes wrong for a forked child, whose first stop is an exit.
type syscallInfo struct {
	Op                 uint8
	_                  [3]uint8
	Arch               uint32
	InstructionPointer uint64
	StackPointer       uint64
	// Entry: the syscall number followed by six arguments.
	// Exit: the return value, then a one-byte error flag.
	Words [7]uint64
}

const (
	ptraceGetSyscallInfo = 0x420e
	syscallInfoEntry     = 1
	syscallInfoExit      = 2
	auditArchX8664       = 0xc000003e

	sysOpen           = 2
	sysTruncate       = 76
	sysRename         = 82
	sysCreat          = 85
	sysLink           = 86
	sysUnlink         = 87
	sysSymlink        = 88
	sysOpenat         = 257
	sysUnlinkat       = 263
	sysRenameat       = 264
	sysLinkat         = 265
	sysSymlinkat      = 266
	sysOpenByHandleAt = 304
	sysRenameat2      = 316
	sysIoUringSetup   = 425
	sysOpenat2        = 437
	sysClone          = 56
	sysPivotRoot      = 155
	sysChroot         = 161
	sysMount          = 165
	sysUnshare        = 272
	sysSetns          = 308
	sysClone3         = 435
	cloneUntraced     = 0x00800000

	openPathOnly = 0x200000 // O_PATH
)

// pendingCall is what an entry stop saw; the exit stop decides how it counts.
type pendingCall struct {
	opens bool
	// read and write are paths the call touches when it succeeds without
	// returning a descriptor (rename, link, unlink, truncate).
	read, write, remove []string
	path                string
	flags               uint64
}

func (o *observation) syscallStop(pid int, state *tracee) {
	var info syscallInfo
	if _, _, errno := syscall.Syscall6(syscall.SYS_PTRACE, ptraceGetSyscallInfo, uintptr(pid),
		unsafe.Sizeof(info), uintptr(unsafe.Pointer(&info)), 0, 0); errno != 0 {
		if errno == syscall.ESRCH {
			// The task was killed between its stop and this query (another
			// thread called exit_group). If it was inside an open, whether the
			// open succeeded is unknown; it is counted as if it did.
			if call := state.pending; call != nil {
				o.limits.OutcomeUnknown++
				if state.image == "" {
					state.image = "(task exited)"
				}
				for _, path := range append(append(append([]string{call.path}, call.read...), call.write...), call.remove...) {
					if path != "" {
						o.use(path, state).Read++
					}
				}
				state.pending = nil
			}
			return
		}
		// An old kernel or a restricted sandbox refuses the query. Staying
		// silent would leave an empty record that looks like a clean one.
		o.traceFailure("the kernel refused PTRACE_GET_SYSCALL_INFO: " + errno.Error())
		return
	}
	switch info.Op {
	case syscallInfoEntry:
		state.pending = nil
		if info.Arch != auditArchX8664 {
			// A 32-bit call has other numbers; reading it with this table would
			// record the wrong thing, so it is counted as not understood.
			o.limits.ForeignABISyscalls++
			return
		}
		arg := func(i int) uint64 { return info.Words[1+i] }
		text := func(i int) string {
			value, ok := readString(pid, arg(i))
			if !ok {
				o.traceFailure(fmt.Sprintf("syscall %d: path argument %d at %#x could not be read", info.Words[0], i, arg(i)))
			}
			return value
		}
		at := func(dirfd, path int) string { return resolve(pid, int(int32(arg(dirfd))), text(path)) }
		here := func(path int) string { return resolve(pid, atFdcwd, text(path)) }
		switch info.Words[0] {
		case sysOpen:
			state.pending = &pendingCall{opens: true, path: here(0), flags: arg(1)}
		case sysCreat:
			state.pending = &pendingCall{opens: true, path: here(0), flags: uint64(syscall.O_WRONLY | syscall.O_CREAT | syscall.O_TRUNC)}
		case sysOpenat:
			state.pending = &pendingCall{opens: true, path: at(0, 1), flags: arg(2)}
		case sysOpenat2:
			// The flags are the first field of the struct open_how argument.
			state.pending = &pendingCall{opens: true, path: at(0, 1), flags: readWord(pid, arg(2))}
		case sysTruncate:
			state.pending = &pendingCall{write: []string{here(0)}}
		case sysUnlink:
			state.pending = &pendingCall{remove: []string{here(0)}}
		case sysUnlinkat:
			state.pending = &pendingCall{remove: []string{at(0, 1)}}
		case sysRename:
			state.pending = &pendingCall{write: []string{here(0), here(1)}}
		case sysRenameat, sysRenameat2:
			state.pending = &pendingCall{write: []string{at(0, 1), at(2, 3)}}
		case sysLink:
			state.pending = &pendingCall{read: []string{here(0)}, write: []string{here(1)}}
		case sysLinkat:
			state.pending = &pendingCall{read: []string{at(0, 1)}, write: []string{at(2, 3)}}
		case sysSymlink:
			state.pending = &pendingCall{write: []string{here(1)}}
		case sysSymlinkat:
			state.pending = &pendingCall{write: []string{at(1, 2)}}
		case sysOpenByHandleAt:
			o.unobservable("open_by_handle_at")
		case sysIoUringSetup:
			o.unobservable("io_uring_setup")
		// Calls that change what a path means, or take a task out of the trace.
		case sysUnshare:
			o.unobservable("unshare")
		case sysSetns:
			o.unobservable("setns")
		case sysMount:
			o.unobservable("mount")
		case sysChroot:
			o.unobservable("chroot")
		case sysPivotRoot:
			o.unobservable("pivot_root")
		case sysClone:
			if arg(0)&cloneUntraced != 0 {
				o.unobservable("clone(CLONE_UNTRACED)")
			}
		case sysClone3:
			if readWord(pid, arg(0))&cloneUntraced != 0 {
				o.unobservable("clone3(CLONE_UNTRACED)")
			}
		}
	case syscallInfoExit:
		call := state.pending
		state.pending = nil
		if call == nil {
			return
		}
		result := int64(info.Words[0])
		if state.image == "" {
			state.image, _ = os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
		}
		if !call.opens {
			if result < 0 {
				return
			}
			for _, path := range call.read {
				o.use(path, state).Read++
			}
			for _, path := range call.write {
				o.use(path, state).Write++
			}
			for _, path := range call.remove {
				o.use(path, state).Remove++
			}
			return
		}
		if result < 0 {
			o.use(call.path, state).Failed++
			return
		}
		// What was opened is what the kernel resolved, not what was asked for:
		// a symbolic link, "..", or /proc/self/root in the request all end here.
		path := call.path
		if resolved := openedPath(pid, int(result)); resolved != "" {
			path = resolved
		}
		use := o.use(path, state)
		access := call.flags & uint64(syscall.O_ACCMODE)
		writes := access != uint64(syscall.O_RDONLY) || call.flags&uint64(syscall.O_CREAT|syscall.O_TRUNC) != 0
		if writes && call.flags&openPathOnly == 0 {
			use.Write++
		}
		if access != uint64(syscall.O_WRONLY) || call.flags&openPathOnly != 0 {
			use.Read++
		}
	}
}

func (o *observation) use(path string, state *tracee) *FileUse {
	key := fileKey{path: path, image: state.image, step: state.step}
	use := o.uses[key]
	if use == nil {
		use = &FileUse{Path: path, Image: state.image, Step: state.step}
		o.uses[key] = use
	}
	return use
}

func (o *observation) traceFailure(what string) {
	o.limits.TraceFailures++
	if len(o.limits.TraceFailureSamples) < 5 {
		o.limits.TraceFailureSamples = append(o.limits.TraceFailureSamples, what)
	}
}

func (o *observation) unobservable(name string) {
	if o.limits.UnobservableCalls == nil {
		o.limits.UnobservableCalls = map[string]int{}
	}
	o.limits.UnobservableCalls[name]++
}

// openedPath asks the kernel what a just-returned descriptor refers to. The
// process's own /proc entries are written back as /proc/self so that they are
// the same path for every process; another process's entries keep its number.
func openedPath(pid, fd int) string {
	target, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/fd/" + strconv.Itoa(fd))
	if err != nil || !strings.HasPrefix(target, "/") {
		return ""
	}
	target = strings.TrimSuffix(target, " (deleted)")
	for _, own := range []int{pid, threadGroupOf(pid)} {
		prefix := "/proc/" + strconv.Itoa(own)
		if own != 0 && (target == prefix || strings.HasPrefix(target, prefix+"/")) {
			return "/proc/self" + target[len(prefix):]
		}
	}
	return target
}

func threadGroupOf(pid int) int {
	tgid, _ := statusNumbers(pid)
	return tgid
}

func readWord(pid int, addr uint64) uint64 {
	mem, err := os.Open("/proc/" + strconv.Itoa(pid) + "/mem")
	if err != nil {
		return 0
	}
	defer mem.Close()
	var buf [8]byte
	if n, _ := mem.ReadAt(buf[:], int64(addr)); n != len(buf) {
		return 0
	}
	return binary.LittleEndian.Uint64(buf[:])
}

// readString reads a NUL-terminated path from the tracee. It reads page by
// page: a read that runs past the end of a mapping fails as a whole, and a
// path near the end of one would be lost.
func readString(pid int, addr uint64) (string, bool) {
	if addr == 0 {
		return "", false
	}
	mem, err := os.Open("/proc/" + strconv.Itoa(pid) + "/mem")
	if err != nil {
		return "", false
	}
	defer mem.Close()
	const page, limit = 4096, 8192
	var text []byte
	for len(text) < limit {
		chunk := make([]byte, page-int((addr+uint64(len(text)))%page))
		n, _ := mem.ReadAt(chunk, int64(addr)+int64(len(text)))
		if n == 0 {
			return "", false
		}
		for i := 0; i < n; i++ {
			if chunk[i] == 0 {
				return string(append(text, chunk[:i]...)), true
			}
		}
		text = append(text, chunk[:n]...)
	}
	return "", false
}

func resolve(pid, dirfd int, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	proc := "/proc/" + strconv.Itoa(pid)
	base := ""
	if dirfd == atFdcwd {
		base, _ = os.Readlink(proc + "/cwd")
	} else {
		base, _ = os.Readlink(proc + "/fd/" + strconv.Itoa(dirfd))
	}
	if base == "" {
		return path
	}
	return filepath.Join(base, path)
}

type imageKey struct {
	dev, ino uint64
	size     int64
	mtime    int64
	ctime    int64
}

var imageDigests = map[imageKey]string{}

// imageDigest hashes the program file behind an exec. The same file is hashed
// once: a compiler that runs hundreds of times is read a single time.
func imageDigest(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	var st syscall.Stat_t
	if err := syscall.Fstat(int(file.Fd()), &st); err != nil {
		return ""
	}
	key := imageKey{dev: st.Dev, ino: st.Ino, size: st.Size, mtime: st.Mtim.Nano(), ctime: st.Ctim.Nano()}
	if digest, ok := imageDigests[key]; ok {
		return digest
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	imageDigests[key] = digest
	return digest
}

// originOf finds the tracee a newly seen task came from: the thread group
// leader for a thread, the parent process for a forked child.
func originOf(pid int, tracees map[int]*tracee) *tracee {
	tgid, ppid := statusNumbers(pid)
	if tgid != 0 && tgid != pid {
		if leader := tracees[tgid]; leader != nil {
			return leader
		}
	}
	return tracees[ppid]
}

// statusNumbers reads a task's thread group and parent process numbers.
func statusNumbers(pid int) (tgid, ppid int) {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if value, ok := strings.CutPrefix(line, "Tgid:"); ok {
			tgid, _ = strconv.Atoi(strings.TrimSpace(value))
		}
		if value, ok := strings.CutPrefix(line, "PPid:"); ok {
			ppid, _ = strconv.Atoi(strings.TrimSpace(value))
		}
	}
	return tgid, ppid
}
