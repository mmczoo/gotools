package utils

import (
	"os"
	"os/exec"
	"path"
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

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//filename, paths
func GenFilePath(filename string, paths ...string) (string, error) {
	dir := path.Join(paths...)

	ok, err := PathExists(dir)
	if err != nil {
		return "", err
	}
	if !ok {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return "", err
		}
	}

	return path.Join(dir, filename), nil
}

func SaveFile(fn string, datas ...[]byte) error {
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, d := range datas {
		f.Write(d)
	}

	return nil
}
