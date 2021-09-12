package utils

import (
	"path/filepath"
	"strings"
)

// returns filename and it's extension
func GetFilenameAndExt(fileName string) (string, string) {
	ext := filepath.Ext(fileName)
	return strings.TrimSuffix(fileName, ext), ext
}
