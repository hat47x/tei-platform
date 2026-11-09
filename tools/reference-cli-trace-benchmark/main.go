// Command reference-cli-trace-benchmark measures where `tei trace` spends time
// and memory on synthetic Workspaces (Issue #1177). It is a measurement tool:
// it does not define product behavior and it never uses customer data.
//
//	go run ./tools/reference-cli-trace-benchmark              # 全寸法・全処理の計測表
//	go run ./tools/reference-cli-trace-benchmark -sizes small # 寸法を絞る
package main

import (
	"bufio"
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
	"sync"
	"time"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/modelref/sourcebridge"
	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/workspace"
)

// ops は end-to-end 計測する処理。TraceWorkspace を CLI と同じ引数で呼ぶ。
var ops = []string{"summary", "full", "full-limit1", "leaf-summary", "leaf-full"}

func hubKey() modelref.Key {
	return modelref.Key{Scope: HubScope, Subject: subject.Subject{Kind: HubKind, ID: HubID}}
}

func leafKey() modelref.Key {
	return modelref.Key{Scope: HubScope, Subject: subject.Subject{Kind: LeafKind, ID: LeafID}}
}

func opArgs(op string) (modelref.Key, modelref.TraceOptions, error) {
	switch op {
	case "summary":
		return hubKey(), modelref.TraceOptions{Mode: "summary"}, nil
	case "full":
		return hubKey(), modelref.TraceOptions{Mode: "full"}, nil
	case "full-limit1":
		return hubKey(), modelref.TraceOptions{Mode: "full", Limit: 1}, nil
	case "leaf-summary":
		return leafKey(), modelref.TraceOptions{Mode: "summary"}, nil
	case "leaf-full":
		return leafKey(), modelref.TraceOptions{Mode: "full"}, nil
	}
	return modelref.Key{}, modelref.TraceOptions{}, fmt.Errorf("unknown op %q", op)
}

type countWriter struct{ n int64 }

func (c *countWriter) Write(p []byte) (int, error) { c.n += int64(len(p)); return len(p), nil }

// e2eResult は子プロセス1回分の結果。VmHWMは子プロセス全体のピーク常駐メモリ。
type e2eResult struct {
	Op          string  `json:"op"`
	Size        string  `json:"size"`
	TraceMS     float64 `json:"traceMs"`
	EncodeMS    float64 `json:"encodeMs"`
	OutputBytes int64   `json:"outputBytes"`
	Total       int     `json:"total"`
	Rows        int     `json:"rows"`
	VmHWMKiB    int64   `json:"vmHWMKiB"`
	AllocMiB    float64 `json:"totalAllocMiB"`
}

func vmHWM() int64 {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return -1
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "VmHWM:") {
			fields := strings.Fields(sc.Text())
			if len(fields) >= 2 {
				v, _ := strconv.ParseInt(fields[1], 10, 64)
				return v
			}
		}
	}
	return -1
}

func runE2E(root, size, op string) error {
	key, options, err := opArgs(op)
	if err != nil {
		return err
	}
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	result, err := operations.TraceWorkspace(root, key, options)
	if err != nil {
		return err
	}
	traceDur := time.Since(start)
	counter := &countWriter{}
	start = time.Now()
	enc := json.NewEncoder(counter)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(result); err != nil {
		return err
	}
	encodeDur := time.Since(start)
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	return json.NewEncoder(os.Stdout).Encode(e2eResult{
		Op: op, Size: size,
		TraceMS: ms(traceDur), EncodeMS: ms(encodeDur), OutputBytes: counter.n,
		Total: result.Trace.Total, Rows: len(result.Trace.Rows),
		VmHWMKiB: vmHWM(), AllocMiB: float64(after.TotalAlloc-before.TotalAlloc) / (1 << 20),
	})
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

type stageResult struct {
	Stage       string  `json:"stage"`
	MS          float64 `json:"ms"`
	AllocMiB    float64 `json:"allocMiB"`
	Mallocs     uint64  `json:"mallocs"`
	PeakHeapMiB float64 `json:"peakHeapAboveBaseMiB"`
}

// measure は1段を実行し、時間・確保量・標本化したヒープ最大値を記録する。
// 段の前にGCして基準ヒープを取り、保持中の前段結果は基準に含める。
func measure(name string, fn func() error) (stageResult, error) {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	var mu sync.Mutex
	peak := before.HeapAlloc
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		var m runtime.MemStats
		for {
			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
				runtime.ReadMemStats(&m)
				mu.Lock()
				if m.HeapAlloc > peak {
					peak = m.HeapAlloc
				}
				mu.Unlock()
			}
		}
	}()
	start := time.Now()
	err := fn()
	dur := time.Since(start)
	close(stop)
	<-done
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	mu.Lock()
	if after.HeapAlloc > peak {
		peak = after.HeapAlloc
	}
	p := peak
	mu.Unlock()
	return stageResult{
		Stage: name, MS: ms(dur),
		AllocMiB:    float64(after.TotalAlloc-before.TotalAlloc) / (1 << 20),
		Mallocs:     after.Mallocs - before.Mallocs,
		PeakHeapMiB: float64(p-before.HeapAlloc) / (1 << 20),
	}, err
}

// runStages は TraceWorkspace の内部段を同じ順に個別実行して計測する。
// 段の再現であり、TraceWorkspace本体の計測は runE2E を正とする。
func runStages(root string) error {
	var out []stageResult
	add := func(name string, fn func() error) error {
		r, err := measure(name, fn)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out = append(out, r)
		return nil
	}
	var loaded workspace.Result
	var derived index.Derived
	var projection derivedbridge.Projection
	var sources sourceindex.Index
	if err := add("1 workspace.Load", func() (err error) { loaded, err = workspace.Load(root); return }); err != nil {
		return err
	}
	if err := add("2 index.Build+derivedbridge", func() error {
		derived = index.Build(loaded.World)
		projection = derivedbridge.FromIndex(derived, HubScope)
		return nil
	}); err != nil {
		return err
	}
	key := hubKey()
	for _, c := range []struct {
		name string
		opt  modelref.TraceOptions
	}{
		{"3a TraceDirect summary", modelref.TraceOptions{Mode: "summary"}},
		{"3b TraceDirect full", modelref.TraceOptions{Mode: "full"}},
		{"3c TraceDirect full limit=1", modelref.TraceOptions{Mode: "full", Limit: 1}},
	} {
		opt := c.opt
		if err := add(c.name, func() error {
			_, err := modelref.TraceDirect(projection.Graph, projection.Rules, key, opt)
			return err
		}); err != nil {
			return err
		}
	}
	if err := add("4 workspace.BuildSourceIndex", func() (err error) { sources, err = workspace.BuildSourceIndex(root, loaded); return }); err != nil {
		return err
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	reader := func(raw string) ([]byte, error) {
		p, err := sourceindex.NormalizeWorkspacePath(raw)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(filepath.Join(rootReal, filepath.FromSlash(p)))
	}
	var resolver *sourcebridge.Resolver
	if err := add("5 sourcebridge.NewResolver", func() (err error) {
		resolver, err = sourcebridge.NewResolver(projection.Graph, sources, reader)
		return
	}); err != nil {
		return err
	}
	if err := add("6 Locate x1", func() error { _ = resolver.Locate(key); return nil }); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}

func child(args ...string) ([]byte, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(self, args...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func median(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-child-e2e" {
		if len(os.Args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: -child-e2e root size op")
			os.Exit(2)
		}
		if err := runE2E(os.Args[2], os.Args[3], os.Args[4]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "-child-stages" {
		if err := runStages(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	sizesFlag := flag.String("sizes", "small,medium,large", "計測する寸法（カンマ区切り）")
	repeat := flag.Int("repeat", 5, "各処理の子プロセス実行回数（中央値を報告）")
	keep := flag.String("workspace-dir", "", "合成Workspaceの生成先（空なら一時ディレクトリ）")
	flag.Parse()
	if err := drive(os.Stdout, *sizesFlag, *repeat, *keep); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func drive(w io.Writer, sizesFlag string, repeat int, keep string) error {
	base := keep
	if base == "" {
		dir, err := os.MkdirTemp("", "tei-trace-bench-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
		base = dir
	}
	fmt.Fprintf(w, "# environment: %s/%s, %s, GOMAXPROCS=%d, NumCPU=%d, repeat=%d (median)\n",
		runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.GOMAXPROCS(0), runtime.NumCPU(), repeat)
	for _, name := range strings.Split(sizesFlag, ",") {
		var size *Size
		for i := range Sizes {
			if Sizes[i].Name == strings.TrimSpace(name) {
				size = &Sizes[i]
			}
		}
		if size == nil {
			return fmt.Errorf("unknown size %q", name)
		}
		root := filepath.Join(base, size.Name)
		if err := Generate(root, *size); err != nil {
			return err
		}
		var bytes int64
		_ = filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
			if err == nil && info.Mode().IsRegular() {
				bytes += info.Size()
			}
			return nil
		})
		fmt.Fprintf(w, "\n## %s: files=%d attributes=%d hubIncomingEdges=%d workspaceBytes=%d\n",
			size.Name, size.Files+1, size.AttributeCount(), size.HubEdgeCount(), bytes)
		fmt.Fprintln(w, "\n| op | trace ms | encode ms | output bytes | total | rows | peak RSS MiB | alloc MiB |")
		fmt.Fprintln(w, "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |")
		for _, op := range ops {
			var trace, enc, rss, alloc []float64
			var last e2eResult
			for i := 0; i < repeat; i++ {
				raw, err := child("-child-e2e", root, size.Name, op)
				if err != nil {
					return fmt.Errorf("%s/%s: %w", size.Name, op, err)
				}
				if err := json.Unmarshal(raw, &last); err != nil {
					return err
				}
				trace = append(trace, last.TraceMS)
				enc = append(enc, last.EncodeMS)
				rss = append(rss, float64(last.VmHWMKiB)/1024)
				alloc = append(alloc, last.AllocMiB)
			}
			fmt.Fprintf(w, "| %s | %.1f | %.1f | %d | %d | %d | %.1f | %.1f |\n",
				op, median(trace), median(enc), last.OutputBytes, last.Total, last.Rows, median(rss), median(alloc))
		}
		fmt.Fprintln(w, "\n| stage (hub subject, 1 process, 1 run) | ms | alloc MiB | mallocs | peak heap above base MiB |")
		fmt.Fprintln(w, "| --- | ---: | ---: | ---: | ---: |")
		raw, err := child("-child-stages", root)
		if err != nil {
			return fmt.Errorf("%s stages: %w", size.Name, err)
		}
		var stages []stageResult
		if err := json.Unmarshal(raw, &stages); err != nil {
			return err
		}
		for _, s := range stages {
			fmt.Fprintf(w, "| %s | %.1f | %.1f | %d | %.1f |\n", s.Stage, s.MS, s.AllocMiB, s.Mallocs, s.PeakHeapMiB)
		}
	}
	return nil
}
