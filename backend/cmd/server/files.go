package main

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func copyOptionalDir(src, dst string) error {
	if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(dst, 0o755)
	}
	return copyDir(src, dst)
}

func copySelectedSkills(srcRoot, dstRoot string, skillIDs []string) error {
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return err
	}
	for _, skillID := range skillIDs {
		src := filepath.Join(srcRoot, filepath.Clean(skillID))
		if !strings.HasPrefix(src, filepath.Clean(srcRoot)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid skill id %q", skillID)
		}
		if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err != nil {
			return fmt.Errorf("skill %q must be a standard skill directory with SKILL.md", skillID)
		}
		if err := copyDir(src, filepath.Join(dstRoot, skillID)); err != nil {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func writeTarball(contextDir string) error {
	out, err := os.Create(filepath.Join(filepath.Dir(contextDir), "context.tar"))
	if err != nil {
		return err
	}
	defer out.Close()
	tw := tar.NewWriter(out)
	defer tw.Close()
	return filepath.WalkDir(contextDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(contextDir, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tw.Write(raw)
		return err
	})
}

func repoPath(repo Repository) string {
	if repo.LocalPath != "" {
		return repo.LocalPath
	}
	if strings.HasPrefix(repo.URL, "file://") {
		return strings.TrimPrefix(repo.URL, "file://")
	}
	return repo.URL
}
