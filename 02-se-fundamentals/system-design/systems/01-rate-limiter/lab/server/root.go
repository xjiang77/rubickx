package server

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func LabRoot() (string, error) {
	if root := os.Getenv("LAB_ROOT"); root != "" {
		return filepath.Abs(root)
	}
	_, source, _, ok := runtime.Caller(0)
	if ok {
		root := filepath.Dir(source)
		if filepath.Base(root) == "server" {
			return rootParent(root), nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "system-design", "systems", "01-rate-limiter", "lab")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, nil
		}
		if filepath.Base(dir) == "lab" {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("rate limiter lab root not found; set LAB_ROOT")
}

func rootParent(serverDir string) string {
	return filepath.Clean(filepath.Join(serverDir, ".."))
}

func labRelativePath(path string) string {
	root, err := LabRoot()
	if err != nil {
		return filepath.ToSlash(path)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}
