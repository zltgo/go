package archive

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

var dest = "/tmp/test-generate-path"

func TestGeneratePath(t *testing.T) {
	os.RemoveAll(dest)
	var mp = make(map[string]string)
	os.MkdirAll(filepath.Join(dest, "/dir"), os.ModePerm)

	Convey("generate dir(1)/file", t, func() {
		path := generatePath(mp, dest, "/dir(1)/file")
		So(path, ShouldEqual, dest+"/dir(1)/file")
		So(mp, ShouldHaveLength, 1)
		So(mp["dir(1)"], ShouldEqual, "dir(1)")

		os.MkdirAll(path, os.ModePerm)
	})

	Convey("generate dir", t, func() {
		path := generatePath(mp, dest, "/dir")
		So(path, ShouldEqual, dest+"/dir(2)")
		So(mp["dir"], ShouldEqual, "dir(2)")
		So(mp["dir(1)"], ShouldEqual, "dir(1)")

		os.MkdirAll(path, os.ModePerm)
	})

	Convey("generate dir/file", t, func() {
		path := generatePath(mp, dest, "/dir/file")
		So(path, ShouldEqual, dest+"/dir(2)/file")
		So(mp["dir"], ShouldEqual, "dir(2)")
		So(mp["dir(1)"], ShouldEqual, "dir(1)")

		os.MkdirAll(path, os.ModePerm)
	})

	Convey("generate dir/file1", t, func() {
		path := generatePath(mp, dest, "/dir/file1")
		So(path, ShouldEqual, dest+"/dir(2)/file1")
		So(mp["dir"], ShouldEqual, "dir(2)")
		So(mp["dir(1)"], ShouldEqual, "dir(1)")

		os.MkdirAll(path, os.ModePerm)
	})

	Convey("generate dir again", t, func() {
		path := generatePath(mp, dest, "/dir")
		So(path, ShouldEqual, dest+"/dir(2)")
		So(mp["dir"], ShouldEqual, "dir(2)")
		So(mp["dir(1)"], ShouldEqual, "dir(1)")
	})
}
