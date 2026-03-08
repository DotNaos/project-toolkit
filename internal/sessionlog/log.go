package sessionlog

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/DotNaos/project-toolkit/internal/projectconfig"
)

type Log struct {
	SessionID string
	FilePath  string
	mu        sync.Mutex
	cwd       string
	gitRoot   string
}

type Event struct {
	Source    string      `json:"source"`
	EventType string      `json:"eventType"`
	Level     string      `json:"level"`
	SkillID   string      `json:"skillId,omitempty"`
	Command   string      `json:"command,omitempty"`
	Message   string      `json:"message,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
}

type eventRecord struct {
	Timestamp string      `json:"timestamp"`
	SessionID string      `json:"sessionId"`
	Source    string      `json:"source"`
	EventType string      `json:"eventType"`
	Level     string      `json:"level"`
	CWD       string      `json:"cwd"`
	GitRoot   string      `json:"gitRoot,omitempty"`
	SkillID   string      `json:"skillId,omitempty"`
	Command   string      `json:"command,omitempty"`
	Message   string      `json:"message,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
}

func Create(cwd, gitRoot string, config projectconfig.Config) (*Log, error) {
	sessionID, err := randomSessionID()
	if err != nil {
		return nil, err
	}

	logsDir := resolveLogsDir(cwd, config)
	filePath := filepath.Join(logsDir, fmt.Sprintf("%s-%s.jsonl", formatTimestampForFileName(time.Now()), sessionID))

	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, err
	}

	if err := os.WriteFile(filePath, []byte{}, 0o644); err != nil {
		return nil, err
	}

	return &Log{SessionID: sessionID, FilePath: filePath, cwd: cwd, gitRoot: gitRoot}, nil
}

func (l *Log) Append(event Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	record := eventRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		SessionID: l.SessionID,
		Source:    event.Source,
		EventType: event.EventType,
		Level:     event.Level,
		CWD:       l.cwd,
		GitRoot:   l.gitRoot,
		SkillID:   event.SkillID,
		Command:   event.Command,
		Message:   event.Message,
		Payload:   event.Payload,
	}

	encoded, err := json.Marshal(record)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(l.FilePath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(encoded, '\n'))
	return err
}

func resolveLogsDir(cwd string, config projectconfig.Config) string {
	if config.Logs == nil || config.Logs.Dir == "" {
		return filepath.Join(cwd, projectconfig.DefaultLogsRelativeDir)
	}

	if filepath.IsAbs(config.Logs.Dir) {
		return config.Logs.Dir
	}

	return filepath.Join(cwd, config.Logs.Dir)
}

func randomSessionID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}

func formatTimestampForFileName(value time.Time) string {
	replacer := func(r rune) rune {
		switch r {
		case ':', '.':
			return '-'
		default:
			return r
		}
	}

	return strings.Map(replacer, value.UTC().Format(time.RFC3339Nano))
}
