package skills

import "os"

func osIOWriteFile(path string, b []byte, mode os.FileMode) error {
	return os.WriteFile(path, b, mode)
}
