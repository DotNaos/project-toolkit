package syncer

import (
	"os/exec"
	"path/filepath"

	"github.com/DotNaos/repo-kit/cli/internal/config"
	"github.com/DotNaos/repo-kit/cli/internal/filesystem"
	"github.com/DotNaos/repo-kit/cli/internal/lock"
)

func Sync(target, kitRoot string) error {
	cfg, err := config.Read(target)
	if err != nil {
		return err
	}
	manifest, err := config.ReadManifest(kitRoot, cfg.Manifest)
	if err != nil {
		return err
	}

	entries := make([]lock.FileLock, 0, len(manifest.Files))
	for _, file := range manifest.Files {
		hash, err := filesystem.CopyFile(
			filepath.Join(kitRoot, "assets", file.Source),
			filepath.Join(target, file.Target),
		)
		if err != nil {
			return err
		}
		entries = append(entries, lock.FileLock{Target: file.Target, SHA256: hash})
	}

	return lock.Write(target, lock.Lock{
		Manifest:  manifest.Name,
		SourceURL: cfg.SourceURL,
		KitCommit: gitSHA(kitRoot),
		Files:     entries,
	})
}

func gitSHA(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return string(bytesTrimSpace(out))
}

func bytesTrimSpace(b []byte) []byte {
	start := 0
	for start < len(b) && (b[start] == ' ' || b[start] == '\n' || b[start] == '\t' || b[start] == '\r') {
		start++
	}
	end := len(b)
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\t' || b[end-1] == '\r') {
		end--
	}
	return b[start:end]
}
