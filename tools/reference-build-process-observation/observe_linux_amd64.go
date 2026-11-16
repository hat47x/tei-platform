//go:build linux && amd64

package main

import (
	"crypto/sha256"
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
	sysOpen   = 2
	sysOpenat = 257

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
	path    string
	flags   uint64
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
	uses := map[fileKey]*FileUse{}
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
			}
			tracees[pid] = state
		}
		sig := ws.StopSignal()
		switch {
		case sig == syscall.SIGTRAP|0x80:
			observeSyscall(pid, state, uses)
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
		for _, use := range uses {
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
)

func observeSyscall(pid int, state *tracee, uses map[fileKey]*FileUse) {
	var info syscallInfo
	if _, _, errno := syscall.Syscall6(syscall.SYS_PTRACE, ptraceGetSyscallInfo, uintptr(pid),
		unsafe.Sizeof(info), uintptr(unsafe.Pointer(&info)), 0, 0); errno != 0 {
		return
	}
	switch info.Op {
	case syscallInfoEntry:
		state.path = ""
		switch info.Words[0] {
		case sysOpen:
			state.path = resolve(pid, atFdcwd, readString(pid, info.Words[1]))
			state.flags = info.Words[2]
		case sysOpenat:
			state.path = resolve(pid, int(int32(info.Words[1])), readString(pid, info.Words[2]))
			state.flags = info.Words[3]
		}
	case syscallInfoExit:
		if state.path == "" {
			return
		}
		if state.image == "" {
			state.image, _ = os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
		}
		key := fileKey{path: state.path, image: state.image, step: state.step}
		use := uses[key]
		if use == nil {
			use = &FileUse{Path: state.path, Image: state.image, Step: state.step}
			uses[key] = use
		}
		switch {
		case int64(info.Words[0]) < 0:
			use.Failed++
		case state.flags&uint64(syscall.O_WRONLY|syscall.O_RDWR) != 0:
			use.Write++
		default:
			use.Read++
		}
		state.path = ""
	}
}

func readString(pid int, addr uint64) string {
	mem, err := os.Open("/proc/" + strconv.Itoa(pid) + "/mem")
	if err != nil {
		return ""
	}
	defer mem.Close()
	buf := make([]byte, 4096)
	n, _ := mem.ReadAt(buf, int64(addr))
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return string(buf[:i])
		}
	}
	return string(buf[:n])
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
	key := imageKey{dev: st.Dev, ino: st.Ino, size: st.Size, mtime: st.Mtim.Nano()}
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
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return nil
	}
	tgid, ppid := 0, 0
	for _, line := range strings.Split(string(raw), "\n") {
		if value, ok := strings.CutPrefix(line, "Tgid:"); ok {
			tgid, _ = strconv.Atoi(strings.TrimSpace(value))
		}
		if value, ok := strings.CutPrefix(line, "PPid:"); ok {
			ppid, _ = strconv.Atoi(strings.TrimSpace(value))
		}
	}
	if tgid != 0 && tgid != pid {
		if leader := tracees[tgid]; leader != nil {
			return leader
		}
	}
	return tracees[ppid]
}
