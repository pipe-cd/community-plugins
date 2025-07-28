package deployment

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func findFirstSQLFile(appDir string) (string, error) {
	var firstSQLFile string
	err := filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".sql") {
			firstSQLFile = path
			return filepath.SkipDir // Stop walking after finding the first .sql file
		}
		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipDir) {
		return "", err
	}
	if firstSQLFile == "" {
		return "", os.ErrNotExist
	}
	return firstSQLFile, nil
}
