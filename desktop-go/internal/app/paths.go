package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func ResolveDataDir() (string, error) {
	if explicit := os.Getenv("DEEP_STUDENT_DATA_DIR"); explicit != "" {
		return ensureDir(explicit)
	}

	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("LOCALAPPDATA")
	case "darwin":
		home, _ := os.UserHomeDir()
		if home != "" {
			base = filepath.Join(home, "Library", "Application Support")
		}
	default:
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			home, _ := os.UserHomeDir()
			if home != "" {
				base = filepath.Join(home, ".local", "share")
			}
		}
	}

	if base == "" {
		base = os.TempDir()
	}

	return ensureDir(filepath.Join(base, "Deep Student"))
}

func LegacyDataDirCandidates(currentDataDir string) []string {
	if os.Getenv("DEEP_STUDENT_DATA_DIR") != "" {
		return nil
	}

	roots := []string{}
	seen := map[string]bool{}
	appendRoot := func(root string) {
		root = strings.TrimSpace(root)
		if root == "" {
			return
		}
		cleaned := filepath.Clean(root)
		key := strings.ToLower(cleaned)
		if key == strings.ToLower(filepath.Clean(currentDataDir)) || seen[key] {
			return
		}
		seen[key] = true
		roots = append(roots, cleaned)
	}
	appendAppRoots := func(base string) {
		if strings.TrimSpace(base) == "" {
			return
		}
		appendRoot(filepath.Join(base, "com.deepstudent.app"))
		appendRoot(filepath.Join(base, "DeepStudent"))
	}

	switch runtime.GOOS {
	case "windows":
		appendAppRoots(os.Getenv("APPDATA"))
		appendAppRoots(os.Getenv("LOCALAPPDATA"))
	case "darwin":
		home, _ := os.UserHomeDir()
		if home != "" {
			appendAppRoots(filepath.Join(home, "Library", "Application Support"))
		}
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			appendAppRoots(xdg)
		}
		home, _ := os.UserHomeDir()
		if home != "" {
			appendAppRoots(filepath.Join(home, ".local", "share"))
		}
	}
	return roots
}

func ensureDir(dir string) (string, error) {
	cleaned := filepath.Clean(dir)
	if err := os.MkdirAll(cleaned, 0o700); err != nil {
		return "", err
	}
	return cleaned, nil
}
