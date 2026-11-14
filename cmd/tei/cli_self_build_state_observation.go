package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// This file records what the entry itself saw of the state directories and of the
// process it started (SB-C0 §4.55).
//
// §4.37 scoped condition 5's "state separation" to what the check compares: the
// string this entry passed and the string the child reported. A child that hands
// the passed value back while building somewhere else satisfies it. That is a
// property of the check, not a finding about the bundled builders, which use the
// directory they are given.
//
// The observation here does not ask the child anything. It looks at the two
// directories before and after each run. It can tell "each run left new files in
// its own directory" from "nothing this entry can see says the directory was
// used"; it cannot tell that no state was shared, because a run can copy or reach
// state this entry does not watch. That limit is written into every record instead
// of being left for a reader to assume, and none of this changes l2b.claimed.

const (
	// stateObservationEffectObserved: each run left new files in the directory it
	// was passed, and the two directories are different objects.
	stateObservationEffectObserved = "effect-observed"
	// stateObservationDeclaredOnly: the entry saw nothing that shows the passed
	// directories were used, so separation rests on the child's report alone.
	stateObservationDeclaredOnly = "declared-only"
)

const stateObservationBoundary = "what this entry saw in the two state directories it passed, without asking the child. effect-observed means each run left new files in its own directory and the two are different objects; it does not show that no state was shared, because writes elsewhere and reads of the other run's state are not watched. declared-only means nothing this entry saw shows the directories were used, so the separation the reproduction rests on is the child's report. It does not change l2b.claimed, and it is not isolation, hermeticity or acceptance"

const stateObservationNoBoundary = "writes outside the passed state directory and reads of the other run's state were not observed: this entry sets up no OS boundary, so a run that shared or copied state elsewhere is not excluded"

// stateDirObservation is one directory, before and after the run it was passed to.
type stateDirObservation struct {
	Path          string `json:"path"`
	ExistedBefore bool   `json:"existedBefore"`
	FilesBefore   int    `json:"filesBefore"`
	ExistsAfter   bool   `json:"existsAfter"`
	FilesAfter    int    `json:"filesAfter"`
	BytesAfter    int64  `json:"bytesAfter"`
	// ReadError is set when the directory could not be walked, so a failed read is
	// never taken for an empty directory.
	ReadError string `json:"readError,omitempty"`
}

// selfBuildStateObservation is the entry's own reading of the two state
// directories. Kind is the only summary; Findings and NotObserved say why.
type selfBuildStateObservation struct {
	Kind   string              `json:"kind"`
	First  stateDirObservation `json:"first"`
	Second stateDirObservation `json:"second"`
	// SameDirectory is true when both paths name one object on disk.
	SameDirectory bool `json:"sameDirectory"`
	// Findings are places where what the entry saw does not support, or
	// contradicts, what the child reported.
	Findings []string `json:"findings,omitempty"`
	// NotObserved is never empty: it names what this entry did not watch.
	NotObserved []string `json:"notObserved"`
	Boundary    string   `json:"boundary"`
}

// countStateDir reports whether the directory exists and how many regular files
// and bytes it holds. A missing directory is (false, 0, 0, nil).
func countStateDir(path string) (exists bool, files int, size int64, err error) {
	info, statErr := os.Stat(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return false, 0, 0, nil
		}
		return false, 0, 0, statErr
	}
	if !info.IsDir() {
		return true, 0, 0, nil
	}
	walkErr := filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			files++
			if fileInfo, infoErr := entry.Info(); infoErr == nil {
				size += fileInfo.Size()
			}
		}
		return nil
	})
	return true, files, size, walkErr
}

// stateDirBefore starts an observation of a directory that is about to be passed
// to a run.
func stateDirBefore(path string) stateDirObservation {
	observation := stateDirObservation{Path: path}
	exists, files, _, err := countStateDir(path)
	observation.ExistedBefore = exists
	observation.FilesBefore = files
	if err != nil {
		observation.ReadError = err.Error()
	}
	return observation
}

// stateDirAfter completes it once the run has ended.
func stateDirAfter(observation stateDirObservation) stateDirObservation {
	exists, files, size, err := countStateDir(observation.Path)
	observation.ExistsAfter = exists
	observation.FilesAfter = files
	observation.BytesAfter = size
	if err != nil && observation.ReadError == "" {
		observation.ReadError = err.Error()
	}
	return observation
}

// sameStateDirectory reports whether two paths name one existing object.
func sameStateDirectory(first, second string) bool {
	firstInfo, err := os.Stat(first)
	if err != nil {
		return false
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		return false
	}
	return os.SameFile(firstInfo, secondInfo)
}

// assessStateObservation turns the two readings into a kind. The reported values
// are the child's; they are used only to name a contradiction, never to support
// the kind.
func assessStateObservation(first, second stateDirObservation, same bool, firstReported, secondReported string) selfBuildStateObservation {
	record := selfBuildStateObservation{
		First:         first,
		Second:        second,
		SameDirectory: same,
		NotObserved:   []string{stateObservationNoBoundary},
		Boundary:      stateObservationBoundary,
	}
	supported := true
	finding := func(text string) {
		record.Findings = append(record.Findings, text)
		supported = false
	}
	if same {
		finding("both runs were passed the same directory object")
	}
	for _, run := range []struct {
		name        string
		observation stateDirObservation
		reported    string
	}{
		{"first", first, firstReported},
		{"second", second, secondReported},
	} {
		switch {
		case run.observation.ReadError != "":
			finding("the " + run.name + " run's state directory could not be read: " + run.observation.ReadError)
		case run.observation.FilesBefore > 0:
			finding("the " + run.name + " run's state directory already held " + strconv.Itoa(run.observation.FilesBefore) + " files before the run, so it was not fresh")
		case !run.observation.ExistsAfter:
			if run.reported != "" {
				finding("the child reported " + run.reported + " as the state the " + run.name + " run used, but that directory does not exist after the run")
			} else {
				finding("the " + run.name + " run's state directory does not exist after the run")
			}
		case run.observation.FilesAfter <= run.observation.FilesBefore:
			finding("the " + run.name + " run left no new file in the state directory it was passed, so nothing this entry saw shows it was used")
		}
	}
	record.Kind = stateObservationDeclaredOnly
	if supported {
		record.Kind = stateObservationEffectObserved
	}
	return record
}

// selfBuildProcessImage is the executable image the started process was running
// when this entry looked. -child's digest is taken from the named file before the
// start; the image can differ, most plainly when the file is a script and the
// image is its interpreter. Recorded, never a condition of the claim.
type selfBuildProcessImage struct {
	Path string `json:"path,omitempty"`
	// MatchesExecutable is true when the image is the file -child named.
	MatchesExecutable bool `json:"matchesExecutable"`
	// NotObserved says why there is no reading, so an absent path is never read as
	// agreement.
	NotObserved string `json:"notObserved,omitempty"`
}

// observeProcessImage reads the image of a process this entry just started. It is
// one reading taken right after the start: the process may replace its image
// later, and that is not watched.
func observeProcessImage(pid int, executable string) *selfBuildProcessImage {
	if runtime.GOOS != "linux" {
		return &selfBuildProcessImage{NotObserved: "the process image is read from /proc, which this entry only does on linux (this host is " + runtime.GOOS + ")"}
	}
	image, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "exe"))
	if err != nil {
		return &selfBuildProcessImage{NotObserved: "the process image could not be read: " + err.Error()}
	}
	record := &selfBuildProcessImage{Path: image}
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		resolved = executable
	}
	record.MatchesExecutable = filepath.Clean(image) == filepath.Clean(resolved)
	return record
}
