package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

func CopyFile(source, target string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	src, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	h := sha256.New()
	writer := io.MultiWriter(dst, h)
	if _, err := io.Copy(writer, src); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
