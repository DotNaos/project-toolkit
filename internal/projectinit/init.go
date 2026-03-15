package projectinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DotNaos/project-toolkit/internal/projectconfig"
)

type Result struct {
	Created []string
	Updated []string
	Skipped []string
}

func Initialize(cwd string, force bool) (Result, error) {
	toolkitDir := filepath.Join(cwd, projectconfig.ProjectToolkitDirName)
	configPath := filepath.Join(cwd, projectconfig.ConfigRelativePath)
	baseWorkspacePath := filepath.Join(cwd, projectconfig.BaseWorkspaceRelativePath)
	projectName := filepath.Base(cwd)

	if err := os.MkdirAll(toolkitDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("failed to create %s: %w", projectconfig.ProjectToolkitDirName, err)
	}

	result := Result{
		Created: []string{},
		Updated: []string{},
		Skipped: []string{},
	}

	if err := writeScaffoldFile(cwd, configPath, buildConfigTemplate(projectName), force, &result); err != nil {
		return Result{}, err
	}

	if err := writeScaffoldFile(cwd, baseWorkspacePath, buildBaseWorkspaceTemplate(), force, &result); err != nil {
		return Result{}, err
	}

	return result, nil
}

func writeScaffoldFile(cwd, filePath, content string, force bool, result *Result) error {
	relativePath, err := filepath.Rel(cwd, filePath)
	if err != nil || relativePath == "" {
		relativePath = filepath.Base(filePath)
	}

	_, statErr := os.Stat(filePath)
	exists := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return fmt.Errorf("failed to inspect %s: %w", relativePath, statErr)
	}

	if exists && !force {
		result.Skipped = append(result.Skipped, relativePath)
		return nil
	}

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", relativePath, err)
	}

	if exists {
		result.Updated = append(result.Updated, relativePath)
		return nil
	}

	result.Created = append(result.Created, relativePath)
	return nil
}

func buildConfigTemplate(projectName string) string {
	return fmt.Sprintf(`project:
  name: %s

dev:
  # args: [npm, run, dev]
  # command: npm run dev
  # router:
  #   mode: portless
  #   name: %s
  #
  # Docker / Compose:
  # args: [docker, compose, -f, compose.yml, up]
  # router:
  #   mode: dockportless
  #   name: %s

logs:
  dir: %s

workspace:
  baseFile: %s

shared:
  # Share gitignored files from the main repo into generated worktrees.
  # source/target default to path when omitted.
  # include/exclude match worktree names.
  - path: .env
`, projectName, projectName, projectName, projectconfig.DefaultLogsRelativeDir, projectconfig.BaseWorkspaceRelativePath)
}

func buildBaseWorkspaceTemplate() string {
	return "{\n  \"folders\": [],\n  \"settings\": {}\n}\n"
}
