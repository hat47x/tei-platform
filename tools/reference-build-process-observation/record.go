package main

const observationSchema = "tei.reference.build-process-observation/v0"

// Exec is one successful execve as the kernel reported it.
type Exec struct {
	Seq int `json:"seq"`
	// ParentSeq is the exec whose program was running in this process (or in
	// the process it was forked from) just before this exec. The command itself
	// has seq 0 and parentSeq -1.
	ParentSeq int `json:"parentSeq"`
	// Step is the exec, directly under the command, that this exec descends
	// from. Everything one child of the command started shares one step.
	Step      int    `json:"step"`
	PID       int    `json:"pid"`
	ParentPID int    `json:"parentPid"`
	Image     string `json:"image"`
	// ImageDigest is the sha256 of the file the kernel mapped as the program,
	// read through /proc/<pid>/exe at the exec stop, not through the path.
	ImageDigest string   `json:"imageDigest,omitempty"`
	Argv        []string `json:"argv"`
	Cwd         string   `json:"cwd"`
	EnvNames    []string `json:"envNames"`
}

// FileUse counts opens of one path by one program image. Read and write are
// what the open flags asked for, not what was later done with the descriptor.
type FileUse struct {
	Path string `json:"path"`
	// Image is the program that was running in the process when it opened the
	// path. A child that has not executed anything yet runs its parent's image.
	Image string `json:"image"`
	// Step is the direct child of the command under which the open happened;
	// 0 means the command itself opened the path.
	Step   int `json:"step"`
	Read   int `json:"read"`
	Write  int `json:"write"`
	Failed int `json:"failed"`
}

type Record struct {
	Schema      string    `json:"schema"`
	Command     []string  `json:"command"`
	ExitCode    int       `json:"exitCode"`
	FilesTraced bool      `json:"filesTraced"`
	Execs       []Exec    `json:"execs"`
	Files       []FileUse `json:"files,omitempty"`
	NotObserved []string  `json:"notObserved"`
}
