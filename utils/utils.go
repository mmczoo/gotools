package utils

import (
	"os"
	"os/exec"
	"path/filepath"
)

func Time33(buf string) int64 {
	h := int64(0)
	for _, c := range buf {
		h *= 33
		h += int64(c)
	}
	if h < 0 {
		return -1 * h
	}
	return h
}

func GetCWD() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	return path
	//return filepath.Dir(path)
}
