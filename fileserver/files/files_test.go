package files

import (
	"regexp"
	"testing"
	"time"
)

func TestUniquePath(t *testing.T) {
	t.Log(UniquePath("./testfile/foo"))
	t.Log(UniquePath("./testfile/.foo"))
	t.Log(UniquePath("./testfile/foo."))
	t.Log(UniquePath("./testfile/foo.bar"))
	t.Log(UniquePath("./testfile/foo.bar."))
	t.Log(UniquePath("./testfile/.foo.bar"))
}

func TestUpdate(t *testing.T) {
	fs, _ := NewRootFiles("./testfile")
	time.Sleep(1 * time.Second)
	for _, fi := range fs.fileList {
		t.Log(fi.Path, "            ", fi.Name)
	}
}

func TestViewPath(t *testing.T) {
	rfs, _ := NewRootFiles("./testfile")
	fis, _ := rfs.ViewPath("/")
	for _, fi := range fis {
		t.Log(fi.Path, "          ", fi.Name)
	}
}

func TestSearchPath(t *testing.T) {
	rfs, _ := NewRootFiles("./testfile")
	time.Sleep(2 * time.Second)

	rx := regexp.MustCompile("\\w+s.go")
	rv, _ := rfs.SearchPath("/", rx)
	for _, v := range rv {
		t.Log(v)
	}
}
