package archive

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// rename file if fpath does exist.
// a ->  a(1)
// a.b -> a(1).b
// .a -> .a(1)
// a. -> a.(1)
// a.b. -> a(1).b.
// .b.c.d -> .b.c.d(1)
func uniquePath(fpath string) string {
	if !isExist(fpath) {
		return fpath
	}

	dir, name := filepath.Split(fpath)
	prefix := name
	ext := ""
	if index := strings.Index(name, "."); index > 0 && index < len(name)-1 {
		prefix = name[:index]
		ext = name[index:]
	}

	for n := 1; ; n++ {
		fpath = dir + prefix + fmt.Sprintf("(%d)", n) + ext
		if !isExist(fpath) {
			break
		}
	}
	return fpath
}

// generate the path for uncompress.
// if the root dir does exist, uniquePath will called.
func generatePath(mp map[string]string, dest, path string) string {
	path = strings.TrimLeft(filepath.ToSlash(path), "/")

	for k, v := range mp {
		if strings.HasPrefix(path, k) {
			// if dest = "/a", path = "b/c", mp["b"] = "b(1)"
			// it returns "/a/b(1)/c"
			return filepath.Join(dest, strings.Replace(path, k, v, 1))
		}
	}

	index := strings.Index(path, "/")
	if index < 0 {
		// if dest = "/a", path = "b", it returns "/a/b(1)" if "/a/b" does exist.
		rv := uniquePath(filepath.Join(dest, path))

		// save to map, mp["b"] = "b(1)"
		_, mp[path] = filepath.Split(rv)
		return rv
	}

	// if dest = "/a", path = "b/c"
	dir := path[0:index] // dir = "b"
	// uniqueDir = "/a/b(1)" if "/a/b" does exist.
	uniqueDir := uniquePath(filepath.Join(dest, dir))

	// save to map, mp["b"] = "b(1)"
	_, mp[dir] = filepath.Split(uniqueDir)

	// it returns "/a/b(1)/c"
	return filepath.Join(uniqueDir, path[index:])
}

//write a new file to local system, fpath will make dir if not exist.
func writeNewFile(fpath string, r io.Reader, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, perm)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

func writeNewSymbolicLink(fpath string, target string) error {
	if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
		return err
	}

	return os.Symlink(target, fpath)
}

//check the path is entry or not.
//if entry is a dir of the path, will return true
func isEntry(path string, entries []string) bool {
	if len(entries) == 0 {
		return true
	}

	for _, entry := range entries {
		entry = strings.TrimLeft(entry, "/")
		if path == entry {
			return true
		}
		if !strings.HasSuffix(entry, "/") {
			entry += "/"
		}
		if strings.HasPrefix(path, entry) {
			return true
		}
	}

	return false
}

type Size interface {
	Size() int64
}

// get size of seeker
func getSize(r io.Seeker) (size int64, err error) {
	switch v := r.(type) {
	case Size:
		size = v.Size()
	default:
		if size, err = r.Seek(0, io.SeekEnd); err == nil {
			_, err = r.Seek(0, io.SeekStart)
		}
	}
	return
}

// get io.ReaderAt from io.ReadSeeker
func getReaderAt(r io.ReadSeeker) io.ReaderAt {
	switch v := r.(type) {
	case io.ReaderAt:
		return v
	default:
		return readerAt{r}
	}
}

type readerAt struct {
	io.ReadSeeker
}

func (r readerAt) ReadAt(b []byte, off int64) (n int, err error) {
	if _, err := r.Seek(off, 0); err != nil {
		return 0, err
	}
	return r.Read(b)
}
