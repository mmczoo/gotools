package utils

import (
	"fmt"
	"testing"
)

func TestGetCWD(t *testing.T) {
	fmt.Println(GetCWD())
}

func TestGenFilePath(t *testing.T) {
	ok, err := GenFilePath("hello.pkl", "test")
	t.Logf("%v %v\n", ok, err)
}
