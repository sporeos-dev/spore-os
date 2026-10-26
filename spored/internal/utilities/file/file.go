// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package file

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// ExecutablePath appends ".exe" on Windows when the path has no extension,
// since manifests name binaries without it (e.g. "./foundry-alias").
func ExecutablePath(path string) string {
	return executablePathFor(runtime.GOOS, path)
}

func executablePathFor(goos, path string) string {
	if goos != "windows" || path == "" || filepath.Ext(path) != "" {
		return path
	}
	return path + ".exe"
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func IsReadable(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func CalculateChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func Read(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(data), nil
}