package devcmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/DotNaos/project-toolkit/internal/projectconfig"
	"github.com/DotNaos/project-toolkit/internal/repocontext"
	"github.com/DotNaos/project-toolkit/internal/sessionlog"
)

const devWrapperSource = "dev-wrapper"

var composeProjectOverridePattern = regexp.MustCompile(`(^|\s)(-p|--project-name)(\s|=)`)

type RunOptions struct {
	Args        []string
	Config      projectconfig.Config
	RepoContext repocontext.Context
	SessionLog  *sessionlog.Log
	Stdout      io.Writer
	Stderr      io.Writer
	Env         []string
}

type Result struct {
	ExitCode int
}

type resolvedCommandSource string

const (
	commandSourceExplicit      resolvedCommandSource = "explicit"
	commandSourceConfigArgs    resolvedCommandSource = "config-args"
	commandSourceConfigCommand resolvedCommandSource = "config-command"
)

type resolvedCommand struct {
	command        string
	args           []string
	displayCommand string
	shell          bool
	source         resolvedCommandSource
	env            []string
	notes          []string
}

func Run(options RunOptions) (Result, error) {
	resolved, err := resolveCommand(options.Args, options.Config, options.RepoContext)
	if err != nil {
		return Result{}, err
	}

	for _, note := range resolved.notes {
		fmt.Fprintln(options.Stdout, note)
	}

	payload := map[string]any{
		"shell": resolved.shell,
		"args":  resolved.args,
	}
	if len(resolved.env) > 0 {
		payload["env"] = resolved.env
	}

	if err := options.SessionLog.Append(sessionlog.Event{
		Source:    devWrapperSource,
		EventType: "command.started",
		Level:     "info",
		Command:   resolved.displayCommand,
		Message:   "Starting dev command",
		Payload:   payload,
	}); err != nil {
		return Result{}, err
	}

	cmd := exec.Command(resolved.command, resolved.args...)
	cmd.Dir = options.RepoContext.CWD
	cmd.Env = mergeEnv(options.Env, resolved.env)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return Result{}, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	stdoutWriter := newLineLoggingWriter(options.Stdout, options.SessionLog, resolved.displayCommand, "stdout", "info")
	stderrWriter := newLineLoggingWriter(options.Stderr, options.SessionLog, resolved.displayCommand, "stderr", "error")

	if err := cmd.Start(); err != nil {
		_ = options.SessionLog.Append(sessionlog.Event{
			Source:    devWrapperSource,
			EventType: "command.failed",
			Level:     "error",
			Command:   resolved.displayCommand,
			Message:   err.Error(),
		})
		return Result{}, fmt.Errorf("failed to start dev command: %s", resolved.displayCommand)
	}

	var copyWG sync.WaitGroup
	copyWG.Add(2)
	go copyStream(&copyWG, stdoutPipe, stdoutWriter)
	go copyStream(&copyWG, stderrPipe, stderrWriter)

	waitErr := cmd.Wait()
	copyWG.Wait()
	stdoutWriter.Flush()
	stderrWriter.Flush()

	exitCode := 0
	if waitErr != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(waitErr, &exitErr); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return Result{}, waitErr
		}
	}

	level := "info"
	if exitCode != 0 {
		level = "error"
	}
	if err := options.SessionLog.Append(sessionlog.Event{
		Source:    devWrapperSource,
		EventType: "command.completed",
		Level:     level,
		Command:   resolved.displayCommand,
		Message:   fmt.Sprintf("Dev command exited with code %d", exitCode),
		Payload: map[string]any{
			"exitCode": exitCode,
		},
	}); err != nil {
		return Result{}, err
	}

	return Result{ExitCode: exitCode}, nil
}

func resolveCommand(args []string, config projectconfig.Config, repoContext repocontext.Context) (resolvedCommand, error) {
	baseCommand, err := resolveBaseCommand(args, config)
	if err != nil {
		return resolvedCommand{}, err
	}

	if config.Dev == nil || config.Dev.Router == nil {
		return baseCommand, nil
	}

	baseName := deriveRouterBaseName(config, repoContext)
	switch config.Dev.Router.Mode {
	case "portless":
		return buildPortlessCommand(baseCommand, baseName, repoContext), nil
	case "dockportless":
		return buildDockportlessCommand(baseCommand, baseName, repoContext)
	default:
		return resolvedCommand{}, fmt.Errorf("%s.dev.router.mode must be 'portless' or 'dockportless'", projectconfig.ConfigRelativePath)
	}
}

func resolveBaseCommand(args []string, config projectconfig.Config) (resolvedCommand, error) {
	normalizedArgs := args
	if len(normalizedArgs) > 0 && normalizedArgs[0] == "--" {
		normalizedArgs = normalizedArgs[1:]
	}

	if len(normalizedArgs) > 0 {
		return resolvedCommand{
			command:        normalizedArgs[0],
			args:           normalizedArgs[1:],
			displayCommand: strings.Join(normalizedArgs, " "),
			shell:          false,
			source:         commandSourceExplicit,
		}, nil
	}

	if config.Dev != nil && len(config.Dev.Args) > 0 {
		return resolvedCommand{
			command:        config.Dev.Args[0],
			args:           config.Dev.Args[1:],
			displayCommand: strings.Join(config.Dev.Args, " "),
			shell:          false,
			source:         commandSourceConfigArgs,
		}, nil
	}

	if config.Dev != nil && config.Dev.Command != "" {
		return resolvedCommand{
			command:        "sh",
			args:           []string{"-lc", config.Dev.Command},
			displayCommand: config.Dev.Command,
			shell:          true,
			source:         commandSourceConfigCommand,
		}, nil
	}

	return resolvedCommand{}, fmt.Errorf("usage: pkit dev [--] <command...> or configure .project-toolkit/config.yaml dev.command/dev.args")
}

func buildPortlessCommand(command resolvedCommand, baseName string, repoContext repocontext.Context) resolvedCommand {
	notes := []string{
		fmt.Sprintf("Portless base URL: http://%s.localhost:1355", baseName),
	}

	if repoContext.GitBranch != "" {
		notes = append(notes, "Linked worktrees keep the same base name and add the branch as a subdomain automatically.")
	}

	if command.source == commandSourceConfigCommand {
		wrappedCommand := fmt.Sprintf("portless run --name %s %s", shellQuote(baseName), command.displayCommand)
		return resolvedCommand{
			command:        "sh",
			args:           []string{"-lc", wrappedCommand},
			displayCommand: wrappedCommand,
			shell:          true,
			source:         command.source,
			notes:          notes,
		}
	}

	args := append([]string{"run", "--name", baseName, command.command}, command.args...)
	return resolvedCommand{
		command:        "portless",
		args:           args,
		displayCommand: "portless " + strings.Join(args, " "),
		shell:          false,
		source:         command.source,
		notes:          notes,
	}
}

func buildDockportlessCommand(command resolvedCommand, baseName string, repoContext repocontext.Context) (resolvedCommand, error) {
	if hasComposeProjectOverride(command) {
		return resolvedCommand{}, fmt.Errorf("do not hardcode Docker Compose project names when dev.router.mode is 'dockportless'; project-toolkit manages COMPOSE_PROJECT_NAME automatically")
	}

	if !looksLikeComposeCommand(command) {
		return resolvedCommand{}, fmt.Errorf("dev.router.mode 'dockportless' only supports compose-compatible commands such as 'docker compose ...' or 'docker-compose ...'")
	}

	projectName, err := deriveDockportlessProjectName(baseName, repoContext)
	if err != nil {
		return resolvedCommand{}, err
	}

	notes := []string{
		fmt.Sprintf("Dockportless project: %s", projectName),
		fmt.Sprintf("Compose project: %s", projectName),
		fmt.Sprintf("Service URLs: http://<service>.%s.localhost:7355", projectName),
	}

	env := []string{"COMPOSE_PROJECT_NAME=" + projectName}
	if command.source == commandSourceConfigCommand {
		wrappedCommand := fmt.Sprintf("COMPOSE_PROJECT_NAME=%s dockportless run %s %s", shellQuote(projectName), shellQuote(projectName), command.displayCommand)
		return resolvedCommand{
			command:        "sh",
			args:           []string{"-lc", wrappedCommand},
			displayCommand: wrappedCommand,
			shell:          true,
			source:         command.source,
			env:            env,
			notes:          notes,
		}, nil
	}

	args := append([]string{"run", projectName, command.command}, command.args...)
	return resolvedCommand{
		command:        "dockportless",
		args:           args,
		displayCommand: fmt.Sprintf("COMPOSE_PROJECT_NAME=%s dockportless %s", projectName, strings.Join(args, " ")),
		shell:          false,
		source:         command.source,
		env:            env,
		notes:          notes,
	}, nil
}

func deriveRouterBaseName(config projectconfig.Config, repoContext repocontext.Context) string {
	if config.Dev != nil && config.Dev.Router != nil && config.Dev.Router.Name != "" {
		return slugify(config.Dev.Router.Name)
	}

	if config.Project != nil && config.Project.Name != "" {
		return slugify(config.Project.Name)
	}

	if repoContext.GitRoot != "" {
		return slugify(filepath.Base(repoContext.GitRoot))
	}

	return slugify(filepath.Base(repoContext.CWD))
}

func deriveDockportlessProjectName(baseName string, repoContext repocontext.Context) (string, error) {
	gitRoot := repoContext.GitRoot
	if gitRoot == "" {
		gitRoot = repoContext.CWD
	}

	linked, err := isLinkedWorktree(gitRoot)
	if err != nil {
		return "", err
	}
	if !linked {
		return baseName, nil
	}

	branchSlug := slugify(repoContext.GitBranch)
	if branchSlug != "" && branchSlug != "project" && branchSlug != baseName {
		return baseName + "-" + branchSlug, nil
	}

	worktreeSlug := slugify(filepath.Base(gitRoot))
	if worktreeSlug != "" && worktreeSlug != "project" && worktreeSlug != baseName {
		return baseName + "-" + worktreeSlug, nil
	}

	return baseName + "-worktree", nil
}

func isLinkedWorktree(gitRoot string) (bool, error) {
	info, err := os.Lstat(filepath.Join(gitRoot, ".git"))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to inspect %s: %w", filepath.Join(gitRoot, ".git"), err)
	}

	return info.Mode().IsRegular(), nil
}

func hasComposeProjectOverride(command resolvedCommand) bool {
	if command.source == commandSourceConfigCommand {
		return composeProjectOverridePattern.MatchString(command.displayCommand)
	}

	for _, arg := range command.args {
		if arg == "-p" || arg == "--project-name" || strings.HasPrefix(arg, "--project-name=") {
			return true
		}
	}

	return false
}

func looksLikeComposeCommand(command resolvedCommand) bool {
	if command.source == commandSourceConfigCommand {
		return strings.Contains(command.displayCommand, "docker compose") ||
			strings.Contains(command.displayCommand, "nerdctl compose") ||
			strings.Contains(command.displayCommand, "podman compose") ||
			strings.Contains(command.displayCommand, "docker-compose") ||
			strings.Contains(command.displayCommand, "podman-compose")
	}

	if command.command == "docker-compose" || command.command == "podman-compose" {
		return true
	}

	if command.command == "docker" || command.command == "nerdctl" || command.command == "podman" {
		return len(command.args) > 0 && command.args[0] == "compose"
	}

	return false
}

func mergeEnv(baseEnv, extraEnv []string) []string {
	if len(extraEnv) == 0 {
		return baseEnv
	}

	result := append([]string{}, baseEnv...)
	for _, entry := range extraEnv {
		key, _, found := strings.Cut(entry, "=")
		if !found {
			result = append(result, entry)
			continue
		}

		replaced := false
		for index, existing := range result {
			existingKey, _, existingFound := strings.Cut(existing, "=")
			if existingFound && existingKey == key {
				result[index] = entry
				replaced = true
				break
			}
		}

		if !replaced {
			result = append(result, entry)
		}
	}

	return result
}

func slugify(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return "project"
	}

	var builder strings.Builder
	lastDash := false
	for _, char := range normalized {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
			lastDash = false
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastDash = false
		case !lastDash:
			builder.WriteRune('-')
			lastDash = true
		}
	}

	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "project"
	}

	return result
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

type lineLoggingWriter struct {
	target   io.Writer
	log      *sessionlog.Log
	command  string
	source   string
	level    string
	buffer   string
	bufferMu sync.Mutex
}

func newLineLoggingWriter(target io.Writer, log *sessionlog.Log, command, source, level string) *lineLoggingWriter {
	return &lineLoggingWriter{target: target, log: log, command: command, source: source, level: level}
}

func (w *lineLoggingWriter) Write(p []byte) (int, error) {
	if _, err := w.target.Write(p); err != nil {
		return 0, err
	}

	w.bufferMu.Lock()
	defer w.bufferMu.Unlock()

	w.buffer += string(p)
	for {
		index := strings.Index(w.buffer, "\n")
		if index == -1 {
			break
		}

		line := strings.TrimSuffix(w.buffer[:index], "\r")
		w.buffer = w.buffer[index+1:]
		_ = w.log.Append(sessionlog.Event{
			Source:    w.source,
			EventType: "command.output",
			Level:     w.level,
			Command:   w.command,
			Message:   line,
		})
	}

	return len(p), nil
}

func (w *lineLoggingWriter) Flush() {
	w.bufferMu.Lock()
	defer w.bufferMu.Unlock()

	remainder := strings.TrimRight(w.buffer, "\r\n")
	if remainder == "" {
		w.buffer = ""
		return
	}

	w.buffer = ""
	_ = w.log.Append(sessionlog.Event{
		Source:    w.source,
		EventType: "command.output",
		Level:     w.level,
		Command:   w.command,
		Message:   remainder,
	})
}

func copyStream(wg *sync.WaitGroup, reader io.Reader, writer io.Writer) {
	defer wg.Done()
	buffered := bufio.NewReader(reader)
	_, _ = io.Copy(writer, buffered)
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
