package checker

import (
	"fmt"
	"path/filepath"

	"github.com/DotNaos/repo-kit/cli/internal/filesystem"
	"github.com/DotNaos/repo-kit/cli/internal/lock"
)

func Drift(target string) ([]string, error) {
	l, err := lock.Read(target)
	if err != nil {
		return nil, err
	}
	var drift []string
	for _, file := range l.Files {
		current, err := filesystem.FileSHA256(filepath.Join(target, file.Target))
		if err != nil {
			drift = append(drift, fmt.Sprintf("missing: %s (%v)", file.Target, err))
			continue
		}
		if current != file.SHA256 {
			drift = append(drift, fmt.Sprintf("changed: %s", file.Target))
		}
	}
	return drift, nil
}
