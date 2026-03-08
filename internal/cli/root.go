package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/DotNaos/project-toolkit/internal/auth"
	"github.com/DotNaos/project-toolkit/internal/nodebridge"
	"github.com/DotNaos/project-toolkit/internal/projectconfig"
	"github.com/DotNaos/project-toolkit/internal/projectinit"
	"github.com/DotNaos/project-toolkit/internal/skillcatalog"
	"github.com/DotNaos/project-toolkit/internal/workspace"
	"github.com/DotNaos/project-toolkit/internal/worktree"
	"github.com/spf13/cobra"
)

const resolveWorkingDirectoryErrorMessage = "failed to resolve working directory: %w"

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "pkit",
		Short:         "project-toolkit hybrid CLI frontend",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(
		newSkillsCommand(),
		newProjectCommand(),
		newPlanCommand(),
		newRunCommand(),
		newDevCommand(),
		newAuthCommand(),
		newCompletionCommand(rootCmd),
	)

	return rootCmd
}

func newSkillsCommand() *cobra.Command {
	skillsCmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage bundled skills",
	}

	skillsCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List discovered skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			skillsRoot, err := skillcatalog.ResolveSkillsRoot()
			if err != nil {
				return err
			}

			skills, err := skillcatalog.Discover(skillsRoot)
			if err != nil {
				return err
			}

			printSkillSummaries(cmd.OutOrStdout(), skills)
			return nil
		},
	})

	return skillsCmd
}

func newProjectCommand() *cobra.Command {
	projectCmd := &cobra.Command{
		Use:   "project",
		Short: "Project/workspace/worktree operations",
	}

	projectCmd.AddCommand(newProjectInitCommand())
	projectCmd.AddCommand(newProjectWorkspaceCommand())
	projectCmd.AddCommand(newProjectWorktreeCommand())

	return projectCmd
}

func newProjectInitCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize project-toolkit scaffold files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := resolveWorkingDirectory()
			if err != nil {
				return err
			}

			result, err := projectinit.Initialize(cwd, force)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Initialized project-toolkit scaffold%s.\n", forceSuffix(force))
			printFileGroup(cmd.OutOrStdout(), "Created", result.Created)
			printFileGroup(cmd.OutOrStdout(), "Updated", result.Updated)
			printFileGroup(cmd.OutOrStdout(), "Skipped", result.Skipped)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "rewrite scaffold files even if they already exist")
	return cmd
}

func newProjectWorkspaceCommand() *cobra.Command {
	workspaceCmd := &cobra.Command{
		Use:   "workspace",
		Short: "Generate and manage workspaces",
	}

	var name string
	var root string
	var output string

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a workspace file for a target root",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := runProjectWorkspaceGenerate(name, root, output)
			if err != nil {
				return err
			}

			printProjectWorkspaceGenerateResult(cmd.OutOrStdout(), result)
			return nil
		},
	}

	generateCmd.Flags().StringVar(&name, "name", "", "logical workspace name")
	generateCmd.Flags().StringVar(&root, "root", "", "target repository or worktree root")
	generateCmd.Flags().StringVar(&output, "output", "", "target workspace file path")
	workspaceCmd.AddCommand(generateCmd)

	return workspaceCmd
}

func newProjectWorktreeCommand() *cobra.Command {
	worktreeCmd := &cobra.Command{
		Use:   "worktree",
		Short: "Create and manage worktrees",
	}

	var branch string
	var base string
	var workspace string
	var output string

	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a managed worktree and matching workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := runProjectWorktreeCreate(args[0], branch, base, workspace, output)
			if err != nil {
				return err
			}

			printProjectWorktreeCreateResult(cmd.OutOrStdout(), result)
			return nil
		},
	}

	createCmd.Flags().StringVar(&branch, "branch", "", "override Git branch name")
	createCmd.Flags().StringVar(&base, "base", "", "base ref for new branches")
	createCmd.Flags().StringVar(&workspace, "workspace", "", "override generated workspace name")
	createCmd.Flags().StringVar(&output, "output", "", "override generated workspace file path")
	worktreeCmd.AddCommand(createCmd)

	return worktreeCmd
}

func newPlanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "plan <skill-id>",
		Short: "Plan an agent run through the TypeScript Codex adapter",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return nodebridge.Run([]string{"plan", args[0]})
		},
	}
}

func newRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run <skill-id>",
		Short: "Execute an agent run through the TypeScript Codex adapter",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return nodebridge.Run([]string{"run", args[0]})
		},
	}
}

func newDevCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "dev [--] <command...>",
		Short: "Run a local development command with logging",
		Args:  cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			bridgeArgs := []string{"dev"}
			bridgeArgs = append(bridgeArgs, args...)
			return nodebridge.Run(bridgeArgs)
		},
	}
}

func newAuthCommand() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication helpers",
	}

	authCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show authentication status for the current adapter",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status := auth.GetStatus(environmentMap())
			fmt.Fprintf(cmd.OutOrStdout(), "Auth source: %s\n", status.Source)
			if status.Available {
				fmt.Fprintln(cmd.OutOrStdout(), "Status: available")
				fmt.Fprintln(cmd.OutOrStdout(), "project-toolkit can authenticate to Codex with the configured API key.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Status: missing")
			fmt.Fprintln(cmd.OutOrStdout(), "Set OPENAI_API_KEY to enable plan/run commands.")
			return nil
		},
	})

	return authCmd
}

func newCompletionCommand(rootCmd *cobra.Command) *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completions from the Go command tree",
	}

	completionCmd.AddCommand(&cobra.Command{
		Use:   "zsh",
		Short: "Print zsh completion",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return rootCmd.GenZshCompletion(io.Writer(rootCmd.OutOrStdout()))
		},
	})

	completionCmd.AddCommand(&cobra.Command{
		Use:   "bash",
		Short: "Print bash completion",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return rootCmd.GenBashCompletion(io.Writer(rootCmd.OutOrStdout()))
		},
	})

	completionCmd.AddCommand(&cobra.Command{
		Use:   "bridge-zsh",
		Short: "Print the current TypeScript-generated zsh completion during migration",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return nodebridge.Run([]string{"completion", "zsh"})
		},
	})

	completionCmd.Long = fmt.Sprintf(
		"The Go CLI owns shell completions going forward. During migration, bridge-zsh still exposes the existing TypeScript completion output.",
	)

	return completionCmd
}

func forceSuffix(force bool) string {
	if force {
		return " (force mode)"
	}

	return ""
}

func printFileGroup(writer io.Writer, label string, files []string) {
	if len(files) == 0 {
		return
	}

	fmt.Fprintf(writer, "%s:\n", label)
	for _, file := range files {
		fmt.Fprintf(writer, "- %s\n", file)
	}
}

func runProjectWorkspaceGenerate(name, root, output string) (workspace.GenerateResult, error) {
	cwd, err := resolveWorkingDirectory()
	if err != nil {
		return workspace.GenerateResult{}, err
	}

	config, err := projectconfig.Load(cwd)
	if err != nil {
		return workspace.GenerateResult{}, err
	}

	workspaceName := name
	if workspaceName == "" {
		workspaceName = "default"
	}

	return workspace.Generate(workspace.GenerateOptions{
		CWD:           cwd,
		Config:        config,
		WorkspaceName: workspaceName,
		OutputPath:    output,
		TargetRoot:    root,
	})
}

func printProjectWorkspaceGenerateResult(writer io.Writer, result workspace.GenerateResult) {
	fmt.Fprintf(writer, "Generated workspace: %s\n", result.OutputPath)
	fmt.Fprintf(writer, "Project key: %s\n", result.ProjectKey)
	fmt.Fprintf(writer, "Workspace name: %s\n", result.WorkspaceName)
	fmt.Fprintf(writer, "Workspace root: %s\n", result.TargetRoot)
	fmt.Fprintf(writer, "Base workspace: %s\n", result.BaseWorkspacePath)
	fmt.Fprintf(writer, "Folder entry: %s\n", result.FolderPath)

	if len(result.SharedLinks) == 0 {
		return
	}

	fmt.Fprintln(writer, "Shared links:")
	for _, sharedLink := range result.SharedLinks {
		reason := ""
		if sharedLink.Reason != "" {
			reason = fmt.Sprintf(" (%s)", sharedLink.Reason)
		}

		fmt.Fprintf(writer, "- [%s] %s -> %s%s\n", sharedLink.Status, sharedLink.Path, sharedLink.TargetPath, reason)
	}
}

func runProjectWorktreeCreate(worktreeName, branch, base, workspaceName, output string) (worktree.CreateResult, error) {
	cwd, err := resolveWorkingDirectory()
	if err != nil {
		return worktree.CreateResult{}, err
	}

	config, err := projectconfig.Load(cwd)
	if err != nil {
		return worktree.CreateResult{}, err
	}

	return worktree.Create(worktree.CreateOptions{
		CWD:                 cwd,
		Config:              config,
		WorktreeName:        worktreeName,
		BranchName:          branch,
		BaseRef:             base,
		WorkspaceName:       workspaceName,
		WorkspaceOutputPath: output,
	})
}

func printProjectWorktreeCreateResult(writer io.Writer, result worktree.CreateResult) {
	fmt.Fprintf(writer, "Created worktree: %s\n", result.WorktreePath)
	fmt.Fprintf(writer, "Branch: %s\n", result.BranchName)
	fmt.Fprintf(writer, "Git root: %s\n", result.GitRoot)
	fmt.Fprintf(writer, "Workspace file: %s\n", result.Workspace.OutputPath)

	if len(result.Workspace.SharedLinks) == 0 {
		return
	}

	fmt.Fprintln(writer, "Shared links:")
	for _, sharedLink := range result.Workspace.SharedLinks {
		reason := ""
		if sharedLink.Reason != "" {
			reason = fmt.Sprintf(" (%s)", sharedLink.Reason)
		}

		fmt.Fprintf(writer, "- [%s] %s -> %s%s\n", sharedLink.Status, sharedLink.Path, sharedLink.TargetPath, reason)
	}
}

func printSkillSummaries(writer io.Writer, skills []skillcatalog.Summary) {
	if len(skills) == 0 {
		fmt.Fprintln(writer, "No skills found.")
		return
	}

	rows := buildSkillSummaryRows(skills)
	widthID, widthTitle, widthValid := measureSkillSummaryWidths(rows)

	fmt.Fprintf(writer, "%s  %s  %s  NOTES\n", padRight("ID", widthID), padRight("TITLE", widthTitle), padRight("VALID", widthValid))
	for _, row := range rows {
		fmt.Fprintf(writer, "%s  %s  %s  %s\n", padRight(row.id, widthID), padRight(row.title, widthTitle), padRight(row.valid, widthValid), row.notes)
	}
}

type skillSummaryRow struct {
	id    string
	title string
	valid string
	notes string
}

func buildSkillSummaryRows(skills []skillcatalog.Summary) []skillSummaryRow {
	rows := make([]skillSummaryRow, 0, len(skills))
	for _, skill := range skills {
		rows = append(rows, skillSummaryRow{
			id:    skill.ID,
			title: displaySkillTitle(skill),
			valid: displaySkillValidity(skill),
			notes: displaySkillNotes(skill),
		})
	}

	return rows
}

func measureSkillSummaryWidths(rows []skillSummaryRow) (int, int, int) {
	widthID := len("ID")
	widthTitle := len("TITLE")
	widthValid := len("VALID")

	for _, row := range rows {
		if len(row.id) > widthID {
			widthID = len(row.id)
		}
		if len(row.title) > widthTitle {
			widthTitle = len(row.title)
		}
		if len(row.valid) > widthValid {
			widthValid = len(row.valid)
		}
	}

	return widthID, widthTitle, widthValid
}

func displaySkillTitle(skill skillcatalog.Summary) string {
	if skill.Title == "" {
		return "-"
	}

	return skill.Title
}

func displaySkillValidity(skill skillcatalog.Summary) string {
	if skill.Valid {
		return "yes"
	}

	return "no"
}

func displaySkillNotes(skill skillcatalog.Summary) string {
	if len(skill.Errors) == 0 {
		return "-"
	}

	return strings.Join(skill.Errors, "; ")
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}

	return value + strings.Repeat(" ", width-len(value))
}

func environmentMap() map[string]string {
	result := map[string]string{}
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}

		result[parts[0]] = parts[1]
	}

	return result
}

func resolveWorkingDirectory() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf(resolveWorkingDirectoryErrorMessage, err)
	}

	return cwd, nil
}
