package system

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// find in root
func findFileInProjectRoot(filename string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, filename)); err == nil {
			return dir, nil // Found the project root
		}
		parentDir := filepath.Dir(dir)
		if parentDir == dir { // Reached the root of the filesystem
			break
		}
		dir = parentDir
	}
	return "", os.ErrNotExist // file not found in project root
}

// LoadEnv loads environment variables from a .env file. If the file is not found in the current directory,
// it searches for it in the project root directory.
func LoadEnv(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		rootDir, rootErr := findFileInProjectRoot(filename)
		if rootErr != nil {
			return rootErr // Return error if project root not found
		}

		f, err = os.Open(filepath.Join(rootDir, filename))
		if err != nil {
			return err
		}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip lines that are not in KEY=VALUE format
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if err := os.Setenv(key, value); err != nil {
			return err // Return error if setting environment variable fails
		}
	}
	return scanner.Err() // Return any error encountered during scanning

}
