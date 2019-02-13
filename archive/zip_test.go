// Copyright 2013 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package archive

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestArchive(t *testing.T) {
	symmetricTest(t, ".zip")
	symmetricTest(t, ".tar")
	symmetricTest(t, ".tar.gz")
	symmetricTest(t, ".tar.bz2")
}

func symmetricTest(t *testing.T, ext string) {
	tmp, err := ioutil.TempDir("", "archiver")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	// Test creating archive
	outfile := filepath.Join(tmp, "test"+ext)
	err = Compress(outfile, "testdata")
	if err != nil {
		t.Fatalf("making archive: didn't expect an error, but got: %v", err)
	}

	var expectedFileCount int
	filepath.Walk("testdata", func(fpath string, info os.FileInfo, err error) error {
		expectedFileCount++
		return nil
	})

	// Test extracting archive
	dest := filepath.Join(tmp, "extraction_test")
	os.Mkdir(dest, 0755)
	err = Decompress(outfile, dest)
	if err != nil {
		t.Fatalf("extracting archive: didn't expect an error, but got: %v", err)
	}

	// If outputs equals inputs, we're good; traverse output files
	// and compare file names, file contents, and file count.

	var actualFileCount int
	filepath.Walk(dest, func(fpath string, info os.FileInfo, err error) error {
		if fpath == dest {
			return nil
		}
		actualFileCount++

		origPath, err := filepath.Rel(dest, fpath)
		if err != nil {
			t.Fatalf("%s: Error inducing original file path: %v", fpath, err)
		}

		if info.IsDir() {
			// stat dir instead of read file
			_, err := os.Stat(origPath)
			if err != nil {
				t.Fatalf("%s: Couldn't stat original directory (%s): %v",
					fpath, origPath, err)
			}
			return nil
		}

		expectedFileInfo, err := os.Stat(origPath)
		if err != nil {
			t.Fatalf("%s: Error obtaining original file info: %v", fpath, err)
		}
		expected, err := ioutil.ReadFile(origPath)
		if err != nil {
			t.Fatalf("%s: Couldn't open original file (%s) from disk: %v",
				fpath, origPath, err)
		}

		actualFileInfo, err := os.Stat(fpath)
		if err != nil {
			t.Fatalf("%s: Error obtaining actual file info: %v", fpath, err)
		}
		actual, err := ioutil.ReadFile(fpath)
		if err != nil {
			t.Fatalf("%s: Couldn't open new file from disk: %v", fpath, err)
		}

		if actualFileInfo.Mode() != expectedFileInfo.Mode() {
			t.Fatalf("%s: File mode differed between on disk and compressed",
				expectedFileInfo.Mode().String()+" : "+actualFileInfo.Mode().String())
		}
		if !bytes.Equal(expected, actual) {
			t.Fatalf("%s: File contents differed between on disk and compressed", origPath)
		}

		return nil
	})

	if got, want := actualFileCount, expectedFileCount; got != want {
		t.Fatalf("Expected %d resulting files, got %d", want, got)
	}
}

func TestZip(t *testing.T) {
	Convey("create a zip file and add  files", t, func() {
		Convey("Add a file that does exist and then add again", func() {
			So(Compress("./testdata/test.zip", "./testdata/quote1.txt"), ShouldBeNil)
			So(Compress("./testdata/test.zip", "testdata/quote1.txt"), ShouldBeNil)
		})

		Convey("Add a dir that does exist", func() {
			So(Compress("./testdata//test.zip", "./testdata/proverbs/extra"), ShouldBeNil)
		})

		Convey("Add a dir that does not exist", func() {
			err := Compress("./testdata/test.zip", "./testdata/proverbs/extra2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "lstat ./testdata/proverbs/extra2: no such file or directory")
		})

		Convey("create a zip that does not exist", func() {
			err := Compress("./testdata/a/b/test.zip", "./testdata/proverbs/extra2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "open ./testdata/a/b/test.zip: no such file or directory")
		})

		Convey("Add many files", func() {
			So(Compress("./testdata/test.zip", "./testdata/already-compressed.jpg", "./testdata/quote1.txt", "./testdata/proverbs", "./testdata/他们"), ShouldBeNil)
		})
	})
}

func TestUnzip(t *testing.T) {
	os.RemoveAll("./testdata/testUnzip")
	Convey("decompress a zip file", t, func() {
		Convey("decompress a file entry", func() {
			So(Decompress("./testdata/test.zip", "./testdata/testUnzip", "quote1.txt"), ShouldBeNil)
		})

		Convey("decompress a dir entry", func() {
			So(Decompress("./testdata/test.zip", "./testdata/testUnzip", "/proverbs/extra/"), ShouldBeNil)
		})

		Convey("decompress with nill entres", func() {
			So(Decompress("./testdata/test.zip", "./testdata/testUnzip/all"), ShouldBeNil)
		})

	})
}
