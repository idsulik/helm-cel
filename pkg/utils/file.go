package utils

import (
	"path/filepath"
)

func GetAbsolutePaths(absPath string, files []string) ([]string, error) {
	var absolutePaths []string
	for _, file := range files {
		path := filepath.Join(absPath, file)
		absolutePaths = append(absolutePaths, path)
	}
	return absolutePaths, nil
}
