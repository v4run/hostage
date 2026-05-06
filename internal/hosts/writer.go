package hosts

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

var ErrConflict = errors.New("hosts file was modified externally")

func ReadMtime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func Write(path string, lines []Line, knownMtime time.Time) error {
	current, err := ReadMtime(path)
	if err != nil {
		return err
	}
	if !current.Equal(knownMtime) {
		return ErrConflict
	}

	content := Format(lines)

	tmp, err := os.CreateTemp("", "hostage-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		log.Printf("hostage: atomic rename failed, falling back to direct write: %v", err)
		var origMode os.FileMode = 0644
		if fi, statErr := os.Stat(path); statErr == nil {
			origMode = fi.Mode().Perm()
		}
		if err2 := os.WriteFile(path, []byte(content), origMode); err2 != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("write %s: %w", path, err2)
		}
		os.Remove(tmpPath)
	}

	return nil
}
