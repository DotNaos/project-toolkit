package devcmd

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/DotNaos/project-toolkit/internal/projectconfig"
	"github.com/DotNaos/project-toolkit/internal/repocontext"
	"github.com/DotNaos/project-toolkit/internal/sessionlog"
)

const devWrapperSource = "dev-wrapper"

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

type resolvedCommand struct {
	command        string
	args           []string
	displayCommand string
	shell          bool
}

func Run(options RunOptions) (Result, error) {
	resolved, err := resolveCommand(options.Args, options.Config)
	if err != nil {
		return Result{}, err
	}

	if err := options.SessionLog.Append(sessionlog.Event{
		Source:    devWrapperSource,
		EventType: "command.started",
		Level:     "info",
		Command:   resolved.displayCommand,
		Message:   "Starting dev command",
		Payload: map[string]any{
			"shell": resolved.shell,
			"args":  resolved.args,
		},
	}); err != nil {
		return Result{}, err
	}

	cmd := exec.Command(resolved.command, resolved.args...)
	cmd.Dir = options.RepoContext.CWD
	cmd.Env = options.Env

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

func resolveCommand(args []string, config projectconfig.Config) (resolvedCommand, error) {
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
		}, nil
	}

	if config.Dev != nil && config.Dev.Command != "" {
		return resolvedCommand{
			command:        "sh",
			args:           []string{"-lc", config.Dev.Command},
			displayCommand: config.Dev.Command,
			shell:          true,
		}, nil
	}

	return resolvedCommand{}, fmt.Errorf("usage: pkit dev [--] <command...> or configure .project-toolkit/config.yaml dev.command")
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
