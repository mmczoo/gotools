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

func TestSaveFile(t *testing.T) {
	ok, err := GenFilePath("hello.pkl", "test")
	t.Logf("%v %v\n", ok, err)
	if err != nil {
		t.Error("gen fail!")
	}

	SaveFile(ok, []byte("heeloelao>htmptl"), []byte("\n<--"),
		[]byte("aaaaaaaaaaaa"), []byte("-->"))

}
