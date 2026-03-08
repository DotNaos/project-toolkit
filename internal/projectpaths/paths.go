package projectpaths

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DotNaos/project-toolkit/internal/projectconfig"
)

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func DeriveProjectKey(cwd string, config projectconfig.Config) string {
	baseName := filepath.Base(cwd)
	if config.Project != nil && config.Project.Name != "" {
		baseName = config.Project.Name
	}

	hash := sha1.Sum([]byte(cwd))
	return slugify(baseName) + "-" + hex.EncodeToString(hash[:])[:8]
}

func GetProjectStateRoot(projectKey string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}

	return filepath.Join(homeDir, projectconfig.ProjectToolkitDirName, "projects", projectKey)
}

func GetGeneratedWorkspacePath(projectKey, workspaceName string) string {
	return filepath.Join(GetProjectStateRoot(projectKey), "workspaces", workspaceName+".code-workspace")
}

func GetManagedWorktreePath(projectKey, worktreeName string) string {
	return filepath.Join(GetProjectStateRoot(projectKey), "worktrees", worktreeName)
}

func slugify(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = slugPattern.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return "project"
	}

	return normalized
}
