package repocontext

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var keyFileCandidates = []string{
	"README.md",
	"package.json",
	"tsconfig.json",
	"go.mod",
	"Makefile",
	"Dockerfile",
	".project-toolkit/config.yaml",
}

type FilePreview struct {
	Path    string `json:"path"`
	Preview string `json:"preview"`
}

type Context struct {
	CWD             string        `json:"cwd"`
	GitRoot         string        `json:"gitRoot,omitempty"`
	GitBranch       string        `json:"gitBranch,omitempty"`
	TopLevelEntries []string      `json:"topLevelEntries"`
	FilePreviews    []FilePreview `json:"filePreviews"`
}

func Collect(cwd string) (Context, error) {
	gitRoot := readGitValue(cwd, []string{"rev-parse", "--show-toplevel"})
	gitBranch := readGitValue(cwd, []string{"branch", "--show-current"})
	topLevelEntries, err := readTopLevelEntries(cwd)
	if err != nil {
		return Context{}, err
	}

	filePreviews, err := readFilePreviews(cwd)
	if err != nil {
		return Context{}, err
	}

	return Context{
		CWD:             cwd,
		GitRoot:         gitRoot,
		GitBranch:       gitBranch,
		TopLevelEntries: topLevelEntries,
		FilePreviews:    filePreviews,
	}, nil
}

func readTopLevelEntries(cwd string) ([]string, error) {
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Name() == ".git" || entry.Name() == "node_modules" {
			continue
		}

		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		results = append(results, name)
	}

	sort.Strings(results)
	if len(results) > 40 {
		results = results[:40]
	}

	return results, nil
}

func readFilePreviews(cwd string) ([]FilePreview, error) {
	results := make([]FilePreview, 0, len(keyFileCandidates))
	for _, relativePath := range keyFileCandidates {
		absolutePath := filepath.Join(cwd, relativePath)
		info, err := os.Stat(absolutePath)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}

		content, err := os.ReadFile(absolutePath)
		if err != nil {
			return nil, err
		}

		results = append(results, FilePreview{
			Path:    relativePath,
			Preview: createPreview(string(content)),
		})
	}

	return results, nil
}

func readGitValue(cwd string, args []string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

func createPreview(source string) string {
	lines := strings.Split(source, "\n")
	if len(lines) > 20 {
		lines = lines[:20]
	}

	preview := strings.TrimSpace(strings.Join(lines, "\n"))
	if len(preview) > 1200 {
		return preview[:1200] + "..."
	}

	return preview
}
