package lock

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type FileLock struct {
	Target string `json:"target"`
	SHA256 string `json:"sha256"`
}

type Lock struct {
	Manifest  string     `json:"manifest"`
	SourceURL string     `json:"source_url"`
	KitCommit string     `json:"kit_commit"`
	SyncedAt  string     `json:"synced_at"`
	Files     []FileLock `json:"files"`
}

func Write(target string, data Lock) error {
	data.SyncedAt = time.Now().UTC().Format(time.RFC3339)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(target, ".repo-kit", "lock.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func Read(target string) (Lock, error) {
	path := filepath.Join(target, ".repo-kit", "lock.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return Lock{}, err
	}
	var l Lock
	if err := json.Unmarshal(b, &l); err != nil {
		return Lock{}, err
	}
	return l, nil
}
