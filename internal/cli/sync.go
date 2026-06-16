package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type syncEntry struct {
	src string
	rel string
}

func resolveSyncEntries(sourceRoot string, specs []string) ([]syncEntry, error) {
	var entries []syncEntry
	seen := make(map[string]struct{})

	for _, spec := range specs {
		matches, err := expandSyncSpec(sourceRoot, spec)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			rel, err := syncRelPath(sourceRoot, match)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[rel]; ok {
				continue
			}
			seen[rel] = struct{}{}
			entries = append(entries, syncEntry{src: match, rel: rel})
		}
	}

	return entries, nil
}

func expandSyncSpec(sourceRoot string, spec string) ([]string, error) {
	pattern := spec
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(sourceRoot, pattern)
	}
	pattern = filepath.Clean(pattern)

	if hasGlobMeta(spec) {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid sync pattern %q: %w", spec, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("sync source matched no files: %s", spec)
		}
		return matches, nil
	}

	if _, err := os.Lstat(pattern); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("sync source not found: %s", spec)
		}
		return nil, fmt.Errorf("stat sync source %q: %w", spec, err)
	}
	return []string{pattern}, nil
}

func syncRelPath(sourceRoot string, absPath string) (string, error) {
	rel, err := filepath.Rel(sourceRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("resolve sync path %q: %w", absPath, err)
	}
	if rel == "." || rel == "" || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("sync source must be inside %s: %s", sourceRoot, absPath)
	}
	return rel, nil
}

func syncEntries(entries []syncEntry, destRoot string) error {
	for _, entry := range entries {
		if err := copySyncPath(entry.src, filepath.Join(destRoot, entry.rel), destRoot); err != nil {
			return err
		}
	}
	return nil
}

func copySyncPath(src string, dst string, destRoot string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("stat sync source %q: %w", src, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return copySymlink(src, dst)
	}
	if !info.IsDir() {
		return copyFile(src, dst, info)
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if pathWithin(destRoot, path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("resolve sync path %q: %w", path, err)
		}
		target := filepath.Join(dst, rel)

		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("stat sync source %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return copySymlink(path, target)
		}
		if info.IsDir() {
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return fmt.Errorf("create sync directory %q: %w", target, err)
			}
			return nil
		}
		return copyFile(path, target, info)
	})
}

func copyFile(src string, dst string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create sync directory %q: %w", filepath.Dir(dst), err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open sync source %q: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open sync destination %q: %w", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy sync source %q: %w", src, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close sync destination %q: %w", dst, err)
	}
	return nil
}

func copySymlink(src string, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return fmt.Errorf("read sync symlink %q: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create sync directory %q: %w", filepath.Dir(dst), err)
	}
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("replace sync destination %q: %w", dst, err)
	}
	if err := os.Symlink(target, dst); err != nil {
		return fmt.Errorf("create sync symlink %q: %w", dst, err)
	}
	return nil
}

func hasGlobMeta(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

func pathWithin(base string, path string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}
