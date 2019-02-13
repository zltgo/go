package archive

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTar(t *testing.T) {
	Convey("create a tar file and add  files", t, func() {
		Convey("Add a file that does exist and then add again", func() {
			So(Compress("./testdata/test.tar", "./testdata/quote1.txt"), ShouldBeNil)
			So(Compress("./testdata/test.tar", "testdata/quote1.txt"), ShouldBeNil)
		})

		Convey("Add a dir that does exist", func() {
			So(Compress("./testdata//test.tar", "./testdata/proverbs/extra"), ShouldBeNil)
		})

		Convey("Add a dir that does not exist", func() {
			err := Compress("./testdata/test.tar", "./testdata/proverbs/extra2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "lstat ./testdata/proverbs/extra2: no such file or directory")
		})

		Convey("Add many files", func() {
			So(Compress("./testdata/test.tar", "./testdata/already-compressed.jpg", "./testdata/quote1.txt", "./testdata/proverbs", "./testdata/他们"), ShouldBeNil)
		})
	})
}

func TestUntar(t *testing.T) {
	os.RemoveAll("./testdata/testUntar")
	Convey("decompress a tar file", t, func() {
		Convey("decompress a file entry", func() {
			So(Decompress("./testdata/test.tar", "./testdata/testUntar", "quote1.txt"), ShouldBeNil)
		})

		Convey("decompress a dir entry", func() {
			So(Decompress("./testdata/test.tar", "./testdata/testUntar", "/proverbs/extra/"), ShouldBeNil)
		})

		Convey("decompress with nill entres", func() {
			So(Decompress("./testdata/test.tar", "./testdata/testUntar/all"), ShouldBeNil)
		})

	})
}
