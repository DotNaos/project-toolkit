package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DotNaos/project-toolkit/internal/projectconfig"
	"github.com/DotNaos/project-toolkit/internal/projectpaths"
	"github.com/tailscale/hujson"
)

type GenerateOptions struct {
	CWD           string
	Config        projectconfig.Config
	WorkspaceName string
	OutputPath    string
	TargetRoot    string
}

type SharedLinkResult struct {
	Path       string
	SourcePath string
	TargetPath string
	Status     string
	Reason     string
}

type GenerateResult struct {
	WorkspaceName     string
	ProjectKey        string
	BaseWorkspacePath string
	OutputPath        string
	TargetRoot        string
	FolderPath        string
	SharedLinks       []SharedLinkResult
}

func Generate(options GenerateOptions) (GenerateResult, error) {
	workspaceName, err := normalizeWorkspaceName(options.WorkspaceName)
	if err != nil {
		return GenerateResult{}, err
	}

	projectKey := projectpaths.DeriveProjectKey(options.CWD, options.Config)
	targetRoot := options.CWD
	if strings.TrimSpace(options.TargetRoot) != "" {
		if filepath.IsAbs(options.TargetRoot) {
			targetRoot = options.TargetRoot
		} else {
			targetRoot = filepath.Join(options.CWD, options.TargetRoot)
		}
	}
	targetRoot = filepath.Clean(targetRoot)

	baseWorkspacePath := resolveBaseWorkspacePath(options.CWD, options.Config)
	outputPath := resolveOutputPath(options.CWD, options.OutputPath, projectKey, workspaceName)

	if err := ensureDirectory(targetRoot, "workspace root"); err != nil {
		return GenerateResult{}, err
	}

	baseWorkspace, err := readBaseWorkspace(baseWorkspacePath)
	if err != nil {
		return GenerateResult{}, err
	}

	workspaceDir := filepath.Dir(outputPath)
	relativeFolderPath, err := filepath.Rel(workspaceDir, targetRoot)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("failed to resolve workspace folder entry for %s: %w", targetRoot, err)
	}
	folderPath := toPortablePath(relativeFolderPath)

	baseWorkspace["folders"] = []map[string]string{{"path": folderPath}}

	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return GenerateResult{}, fmt.Errorf("failed to create workspace directory %s: %w", workspaceDir, err)
	}

	formatted, err := json.MarshalIndent(baseWorkspace, "", "  ")
	if err != nil {
		return GenerateResult{}, fmt.Errorf("failed to format workspace file: %w", err)
	}

	if err := os.WriteFile(outputPath, append(formatted, '\n'), 0o644); err != nil {
		return GenerateResult{}, fmt.Errorf("failed to write workspace file %s: %w", outputPath, err)
	}

	sharedLinks, err := applySharedLinks(options.CWD, targetRoot, workspaceName, options.Config.Shared)
	if err != nil {
		return GenerateResult{}, err
	}

	return GenerateResult{
		WorkspaceName:     workspaceName,
		ProjectKey:        projectKey,
		BaseWorkspacePath: baseWorkspacePath,
		OutputPath:        outputPath,
		TargetRoot:        targetRoot,
		FolderPath:        folderPath,
		SharedLinks:       sharedLinks,
	}, nil
}

func normalizeWorkspaceName(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", fmt.Errorf("workspace name must be a non-empty string")
	}

	return normalized, nil
}

func resolveBaseWorkspacePath(cwd string, config projectconfig.Config) string {
	configuredPath := projectconfig.BaseWorkspaceRelativePath
	if config.Workspace != nil && config.Workspace.BaseFile != "" {
		configuredPath = config.Workspace.BaseFile
	}

	if filepath.IsAbs(configuredPath) {
		return configuredPath
	}

	return filepath.Join(cwd, configuredPath)
}

func resolveOutputPath(cwd, outputPath, projectKey, workspaceName string) string {
	if strings.TrimSpace(outputPath) != "" {
		if filepath.IsAbs(outputPath) {
			return outputPath
		}

		return filepath.Join(cwd, outputPath)
	}

	return projectpaths.GetGeneratedWorkspacePath(projectKey, workspaceName)
}

func readBaseWorkspace(filePath string) (map[string]any, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace base file %s: %w", filePath, err)
	}

	standardized, err := hujson.Standardize(source)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace base file %s: %w", filePath, err)
	}

	var obj map[string]any
	if err := json.Unmarshal(standardized, &obj); err != nil {
		return nil, fmt.Errorf("invalid workspace base file %s: %w", filePath, err)
	}

	if obj == nil {
		return nil, fmt.Errorf("workspace base file %s must contain a JSON object", filePath)
	}

	return obj, nil
}

func applySharedLinks(sourceRoot, targetRoot, workspaceName string, entries []projectconfig.SharedLinkConfig) ([]SharedLinkResult, error) {
	results := make([]SharedLinkResult, 0, len(entries))
	for _, entry := range entries {
		result, err := applySharedLink(entry, sourceRoot, targetRoot, workspaceName)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

func applySharedLink(entry projectconfig.SharedLinkConfig, sourceRoot, targetRoot, workspaceName string) (SharedLinkResult, error) {
	resolved := resolveSharedLinkPaths(entry, sourceRoot, targetRoot)

	if !matchesWorkspace(entry, workspaceName) {
		return buildSharedLinkResult(entry.Path, resolved.sourcePath, resolved.targetPath, "skipped", fmt.Sprintf("workspace '%s' is filtered out", workspaceName)), nil
	}

	if resolved.sourcePath == resolved.targetPath {
		return buildSharedLinkResult(entry.Path, resolved.sourcePath, resolved.targetPath, "skipped", "source and target are identical"), nil
	}

	sourceInfo, err := os.Lstat(resolved.sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return buildSharedLinkResult(entry.Path, resolved.sourcePath, resolved.targetPath, "missing-source", "source path does not exist"), nil
		}

		return SharedLinkResult{}, fmt.Errorf("failed to inspect shared link source %s: %w", resolved.sourcePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(resolved.targetPath), 0o755); err != nil {
		return SharedLinkResult{}, fmt.Errorf("failed to create shared link directory for %s: %w", resolved.targetPath, err)
	}

	existing, err := inspectExistingTarget(resolved.targetPath, resolved.sourcePath)
	if err != nil {
		return SharedLinkResult{}, err
	}
	if existing != nil {
		return buildSharedLinkResult(entry.Path, resolved.sourcePath, resolved.targetPath, existing.status, existing.reason), nil
	}

	relativeSourcePath, err := filepath.Rel(filepath.Dir(resolved.targetPath), resolved.sourcePath)
	if err != nil {
		return SharedLinkResult{}, fmt.Errorf("failed to resolve shared link path %s -> %s: %w", resolved.targetPath, resolved.sourcePath, err)
	}
	if relativeSourcePath == "" {
		relativeSourcePath = "."
	}

	linkType := "file"
	if sourceInfo.IsDir() {
		linkType = "dir"
	}

	if err := os.Symlink(relativeSourcePath, resolved.targetPath); err != nil {
		return SharedLinkResult{}, fmt.Errorf("failed to create %s symlink %s -> %s: %w", linkType, resolved.targetPath, relativeSourcePath, err)
	}

	return buildSharedLinkResult(entry.Path, resolved.sourcePath, resolved.targetPath, "linked", ""), nil
}

func resolveSharedLinkPaths(entry projectconfig.SharedLinkConfig, sourceRoot, targetRoot string) struct{ sourcePath, targetPath string } {
	sourcePath := entry.Path
	if entry.Source != "" {
		sourcePath = entry.Source
	}

	targetPath := entry.Path
	if entry.Target != "" {
		targetPath = entry.Target
	}

	return struct{ sourcePath, targetPath string }{
		sourcePath: filepath.Join(sourceRoot, sourcePath),
		targetPath: filepath.Join(targetRoot, targetPath),
	}
}

func inspectExistingTarget(targetPath, sourcePath string) (*struct{ status, reason string }, error) {
	_, err := os.Lstat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to inspect existing shared link target %s: %w", targetPath, err)
	}

	alreadyLinked, err := isSymlinkToTarget(targetPath, sourcePath)
	if err != nil {
		return nil, err
	}

	if alreadyLinked {
		return &struct{ status, reason string }{status: "skipped", reason: "shared link already exists"}, nil
	}

	return &struct{ status, reason string }{status: "conflict", reason: "target path already exists"}, nil
}

func isSymlinkToTarget(linkPath, sourcePath string) (bool, error) {
	info, err := os.Lstat(linkPath)
	if err != nil {
		return false, fmt.Errorf("failed to inspect existing symlink %s: %w", linkPath, err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}

	resolvedLinkPath, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve symlink %s: %w", linkPath, err)
	}

	resolvedSourcePath, err := filepath.EvalSymlinks(sourcePath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve source path %s: %w", sourcePath, err)
	}

	return resolvedLinkPath == resolvedSourcePath, nil
}

func matchesWorkspace(entry projectconfig.SharedLinkConfig, workspaceName string) bool {
	if len(entry.Include) > 0 && !contains(entry.Include, workspaceName) {
		return false
	}

	if contains(entry.Exclude, workspaceName) {
		return false
	}

	return true
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func ensureDirectory(dirPath, label string) error {
	info, err := os.Lstat(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s does not exist: %s", label, dirPath)
		}

		return fmt.Errorf("failed to inspect %s %s: %w", label, dirPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s must be a directory: %s", label, dirPath)
	}

	return nil
}

func buildSharedLinkResult(pathValue, sourcePath, targetPath, status, reason string) SharedLinkResult {
	result := SharedLinkResult{
		Path:       pathValue,
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Status:     status,
	}

	if reason != "" {
		result.Reason = reason
	}

	return result
}

func toPortablePath(value string) string {
	if value == "" {
		return "."
	}

	return filepath.ToSlash(value)
}
