package planner

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func FolderByDate(t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	return filepath.Join(t.Format("2006"), t.Format("2006-01-02"))
}

func ResolveTargetPath(destRoot string, shotAt time.Time, fileName string, exists func(string) bool) string {
	dir := filepath.Join(destRoot, FolderByDate(shotAt))
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	ext := filepath.Ext(fileName)

	initial := filepath.Join(dir, fileName)
	if !exists(initial) {
		return initial
	}

	for i := 1; i <= 9999; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_%03d%s", base, i, ext))
		if !exists(candidate) {
			return candidate
		}
	}

	return filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, time.Now().Unix(), ext))
}
