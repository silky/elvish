package util

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestGetwd(t *testing.T) {
	dir, error := ioutil.TempDir("", "elvishtest.")
	if error != nil {
		t.Errorf("Got error when creating temp dir: %v", error)
	} else {
		os.Chdir(dir)
		if gotwd := Getwd(); gotwd != dir {
			t.Errorf("Getwd() -> %v, want %v", gotwd, dir)
		}
		os.Remove(dir)
	}

	os.Chdir(os.Getenv("HOME"))
	if gotwd := Getwd(); gotwd != "~" {
		t.Errorf("Getwd() -> %v, want ~", gotwd)
	}
}
