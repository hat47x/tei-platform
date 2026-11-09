package exportpackage

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Export categories. A delivery package must say which of these it carries and
// which it deliberately leaves out.
const (
	CategoryMeaning   = "meaning"
	CategoryData      = "data"
	CategoryAuthority = "authority"
	CategoryEvidence  = "evidence"
)

// Categories is the fixed set, in report order.
var Categories = []string{CategoryMeaning, CategoryData, CategoryAuthority, CategoryEvidence}

// Policy selects the categories to carry. An empty Include means every category;
// Exclude always wins. A file outside the known categories is rejected rather
// than exported silently.
type Policy struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// categorize maps a root-relative path to its category by the first path
// segment. A file outside the known directories is unclassified.
func categorize(path string) (string, bool) {
	first := strings.SplitN(filepath.ToSlash(path), "/", 2)[0]
	for _, category := range Categories {
		if first == category {
			return category, true
		}
	}
	return "", false
}

func categorySet(values []string) (map[string]bool, error) {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		known := false
		for _, category := range Categories {
			if value == category {
				known = true
				break
			}
		}
		if !known {
			return nil, fmt.Errorf("unknown export category %q", value)
		}
		set[value] = true
	}
	return set, nil
}

// ExportWithPolicy builds a package that carries only the selected categories and
// records the included, excluded, and empty categories in the manifest.
func ExportWithPolicy(root, out string, policy Policy) (Manifest, error) {
	include, err := categorySet(policy.Include)
	if err != nil {
		return Manifest{}, err
	}
	exclude, err := categorySet(policy.Exclude)
	if err != nil {
		return Manifest{}, err
	}
	if len(include) == 0 {
		for _, category := range Categories {
			include[category] = true
		}
	}
	effective := make(map[string]bool, len(include))
	for category := range include {
		if !exclude[category] {
			effective[category] = true
		}
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return Manifest{}, err
	}
	outAbs, err := filepath.Abs(out)
	if err != nil {
		return Manifest{}, err
	}
	info, err := os.Stat(rootAbs)
	if err != nil || !info.IsDir() {
		return Manifest{}, fmt.Errorf("package root is not a directory: %s", rootAbs)
	}
	relative, err := filepath.Rel(rootAbs, outAbs)
	if err == nil && !strings.HasPrefix(relative, "..") && relative != "." {
		return Manifest{}, fmt.Errorf("package output must be outside the package root")
	}

	var files []File
	seen := make(map[string]bool, len(effective))
	if err := filepath.WalkDir(rootAbs, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("package contains a symbolic link: %s", path)
		}
		relative, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return err
		}
		slashed := filepath.ToSlash(relative)
		category, ok := categorize(slashed)
		if !ok {
			// A file the policy cannot classify is a policy error, not something
			// to export and hope the receiver ignores.
			return fmt.Errorf("package file %q is outside the known categories %v", slashed, Categories)
		}
		if !effective[category] {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		seen[category] = true
		files = append(files, File{
			Path: slashed, Size: int64(len(content)), Digest: digestBytes(content), Category: category,
		})
		return nil
	}); err != nil {
		return Manifest{}, err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	manifest := Manifest{
		Schema:             Schema,
		Files:              files,
		IncludedCategories: sortedCategories(effective),
		ExcludedCategories: sortedCategories(exclude),
		EmptyCategories:    emptyCategories(effective, seen),
	}
	manifest.Digest = manifestDigest(files)

	if err := os.MkdirAll(filepath.Dir(outAbs), 0o755); err != nil {
		return Manifest{}, err
	}
	if err := writePackage(outAbs, manifest, rootAbs); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func sortedCategories(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for _, category := range Categories {
		if set[category] {
			out = append(out, category)
		}
	}
	return out
}

func emptyCategories(effective, seen map[string]bool) []string {
	var out []string
	for _, category := range Categories {
		if effective[category] && !seen[category] {
			out = append(out, category)
		}
	}
	return out
}

// writePackage writes the manifest and the selected files. The caller has
// already validated paths and digests.
func writePackage(outAbs string, manifest Manifest, rootAbs string) error {
	manifestRaw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.Create(outAbs)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(file)
	closeAll := func() error {
		if err := writer.Close(); err != nil {
			file.Close()
			return err
		}
		return file.Close()
	}
	header := &zip.FileHeader{Name: "manifest.json", Method: zip.Deflate}
	header.SetMode(0o644)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		file.Close()
		return err
	}
	if _, err := entry.Write(append(manifestRaw, '\n')); err != nil {
		file.Close()
		return err
	}
	for _, item := range manifest.Files {
		content, err := os.ReadFile(filepath.Join(rootAbs, filepath.FromSlash(item.Path)))
		if err != nil {
			file.Close()
			return err
		}
		itemHeader := &zip.FileHeader{Name: "files/" + item.Path, Method: zip.Deflate}
		itemHeader.SetMode(0o644)
		itemEntry, err := writer.CreateHeader(itemHeader)
		if err != nil {
			file.Close()
			return err
		}
		if _, err := itemEntry.Write(content); err != nil {
			file.Close()
			return err
		}
	}
	return closeAll()
}
