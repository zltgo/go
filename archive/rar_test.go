package archive

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnrar(t *testing.T) {
	os.RemoveAll("./testdata/testUnrar")
	Convey("decompress a rar file", t, func() {
		Convey("decompress a file entry", func() {
			So(Decompress("./testdata/test.rar", "./testdata/testUnrar", "quote1.txt"), ShouldBeNil)
		})

		Convey("decompress a dir entry", func() {
			So(Decompress("./testdata/test.rar", "./testdata/testUnrar", "/proverbs/extra/"), ShouldBeNil)
		})

		Convey("decompress with nill entres", func() {
			So(Decompress("./testdata/test.rar", "./testdata/testUnrar/all"), ShouldBeNil)
		})
	})
}
