package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/DotNaos/project-toolkit/internal/projectconfig"
	"github.com/DotNaos/project-toolkit/internal/projectpaths"
	"github.com/DotNaos/project-toolkit/internal/workspace"
)

type CreateOptions struct {
	CWD                 string
	Config              projectconfig.Config
	WorktreeName        string
	BranchName          string
	BaseRef             string
	WorkspaceName       string
	WorkspaceOutputPath string
}

type CreateResult struct {
	BranchName   string
	GitRoot      string
	WorktreeName string
	WorktreePath string
	Workspace    workspace.GenerateResult
}

func Create(options CreateOptions) (CreateResult, error) {
	worktreeName, err := normalizeName(options.WorktreeName, "worktree name")
	if err != nil {
		return CreateResult{}, err
	}

	branchInput := options.BranchName
	if strings.TrimSpace(branchInput) == "" {
		branchInput = worktreeName
	}
	branchName, err := normalizeName(branchInput, "branch name")
	if err != nil {
		return CreateResult{}, err
	}

	workspaceInput := options.WorkspaceName
	if strings.TrimSpace(workspaceInput) == "" {
		workspaceInput = worktreeName
	}
	workspaceName, err := normalizeName(workspaceInput, "workspace name")
	if err != nil {
		return CreateResult{}, err
	}

	gitRoot, err := resolveGitRoot(options.CWD)
	if err != nil {
		return CreateResult{}, err
	}

	projectKey := projectpaths.DeriveProjectKey(options.CWD, options.Config)
	worktreePath := projectpaths.GetManagedWorktreePath(projectKey, worktreeName)

	if err := ensurePathAvailable(worktreePath, "worktree path"); err != nil {
		return CreateResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return CreateResult{}, fmt.Errorf("failed to create worktree parent directory %s: %w", filepath.Dir(worktreePath), err)
	}

	branchExists, err := gitBranchExists(gitRoot, branchName)
	if err != nil {
		return CreateResult{}, err
	}

	if err := addGitWorktree(addOptions{
		GitRoot:      gitRoot,
		WorktreePath: worktreePath,
		BranchName:   branchName,
		BranchExists: branchExists,
		BaseRef:      strings.TrimSpace(options.BaseRef),
	}); err != nil {
		return CreateResult{}, err
	}

	workspaceResult, err := workspace.Generate(workspace.GenerateOptions{
		CWD:           options.CWD,
		Config:        options.Config,
		WorkspaceName: workspaceName,
		OutputPath:    options.WorkspaceOutputPath,
		TargetRoot:    worktreePath,
	})
	if err != nil {
		cleanupErr := removeGitWorktree(gitRoot, worktreePath)
		if cleanupErr != nil {
			return CreateResult{}, fmt.Errorf("%w (cleanup failed: %v)", err, cleanupErr)
		}

		return CreateResult{}, err
	}

	return CreateResult{
		BranchName:   branchName,
		GitRoot:      gitRoot,
		WorktreeName: worktreeName,
		WorktreePath: worktreePath,
		Workspace:    workspaceResult,
	}, nil
}

type addOptions struct {
	GitRoot      string
	WorktreePath string
	BranchName   string
	BranchExists bool
	BaseRef      string
}

func addGitWorktree(options addOptions) error {
	args := []string{"worktree", "add"}
	if !options.BranchExists {
		args = append(args, "-b", options.BranchName)
	}

	args = append(args, options.WorktreePath)

	if options.BranchExists {
		args = append(args, options.BranchName)
	} else if options.BaseRef != "" {
		args = append(args, options.BaseRef)
	}

	_, err := runGit(options.GitRoot, args)
	return err
}

func removeGitWorktree(gitRoot, worktreePath string) error {
	_, err := runGit(gitRoot, []string{"worktree", "remove", "--force", worktreePath})
	return err
}

func resolveGitRoot(cwd string) (string, error) {
	output, err := runGit(cwd, []string{"rev-parse", "--show-toplevel"})
	if err != nil {
		return "", err
	}

	normalized := strings.TrimSpace(output)
	if normalized == "" {
		return "", fmt.Errorf("current working directory must be inside a Git repository")
	}

	return normalized, nil
}

func ensurePathAvailable(targetPath, label string) error {
	_, err := os.Stat(targetPath)
	if err == nil {
		return fmt.Errorf("%s already exists: %s", label, targetPath)
	}

	if os.IsNotExist(err) {
		return nil
	}

	return fmt.Errorf("failed to inspect %s %s: %w", label, targetPath, err)
}

func gitBranchExists(cwd, branchName string) (bool, error) {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", branchName))
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}

		return false, fmt.Errorf("git command failed: git show-ref --verify --quiet refs/heads/%s (%s)", branchName, getErrorMessage(err))
	}

	return true, nil
}

func runGit(cwd string, args []string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: git %s (%s)", strings.Join(args, " "), formatCommandError(output, err))
	}

	return strings.TrimSpace(string(output)), nil
}

func normalizeName(value, label string) (string, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", fmt.Errorf("%s must be a non-empty string", label)
	}

	return normalized, nil
}

func formatCommandError(output []byte, err error) string {
	text := strings.TrimSpace(string(output))
	if text != "" {
		return text
	}

	return getErrorMessage(err)
}

func getErrorMessage(err error) string {
	if err == nil {
		return "unknown error"
	}

	return err.Error()
}

func asExitError(err error, target **exec.ExitError) bool {
	if err == nil {
		return false
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}

	*target = exitErr
	return true
}
