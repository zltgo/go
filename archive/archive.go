package archive

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//a common interface to tar/zip/gzip/rar
type Writer interface {
	AddEmptyDir(relPath string, perm os.FileMode) error
	AddReader(info os.FileInfo, r io.Reader, relPath string) error
	Close() error
}

type Reader interface {
	ExtractTo(dest string, entries ...string) error
	Close() error
}

type NewWriterFunc func(io.Writer) (Writer, error)
type NewReaderFunc func(io.Reader, io.ReaderAt, int64) (Reader, error)

var (
	//config
	DirMaxSize  int64 = -1 //defaults to no limit (-1)
	DirMaxFiles int   = -1 //defaults to no limit (-1)

	//use when create a archive file
	FileMode = os.ModePerm

	// CompressedFormats is a (non-exhaustive) set of lowercased
	// file extensions for formats that are typically already
	// compressed. Compressing already-compressed files often
	// results in a larger file, so when possible, we check this
	// set to avoid that.
	CompressedFormats = map[string]bool{
		".cbr":  true,
		".cbz":  true,
		".ar":   true,
		".7z":   true,
		".avi":  true,
		".bz2":  true,
		".cab":  true,
		".gif":  true,
		".gz":   true,
		".jar":  true,
		".jpeg": true,
		".jpg":  true,
		".lz":   true,
		".lzma": true,
		".mov":  true,
		".mp3":  true,
		".mp4":  true,
		".mpeg": true,
		".mpg":  true,
		".png":  true,
		".rar":  true,
		".tgz":  true,
		".xz":   true,
		".zip":  true,
		".zipx": true,
	}
	//use Register to support a file type, see zip/tar/gzip/rar
	writerMap = make(map[string]NewWriterFunc)
	readerMap = make(map[string]NewReaderFunc)
)

//not concurrent safe, use in init of your package
func RegisterWriter(ext string, fn NewWriterFunc) {
	writerMap[strings.ToLower(ext)] = fn
}

//not concurrent safe, use in init of your package
func RegisterReader(ext string, fn NewReaderFunc) {
	readerMap[strings.ToLower(ext)] = fn
}

//get a archive.Writer by name
func NewWriter(name string, w io.Writer) (Writer, error) {
	//get file type
	ext := strings.ToLower(filepath.Ext(name))
	fn, ok := writerMap[ext]
	if !ok {
		return nil, errors.New("unknown archive writer name: " + name)
	}

	return fn(w)
}

//get a archive.Reader by name
func NewReader(name string, r io.Reader, ra io.ReaderAt, size int64) (Reader, error) {
	//get file type
	ext := strings.ToLower(filepath.Ext(name))
	fn, ok := readerMap[ext]
	if !ok {
		return nil, errors.New("unknown archive reader name: " + name)
	}

	return fn(r, ra, size)
}

// decompress the src file  into dest.
// if entries is not nil,  only the file or dir in entries will decompress.
func Decompress(src, dest string, entries ...string) error {
	srcf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcf.Close()

	return DecompressReader(src, srcf, dest, entries...)
}

// decompress the r as src file  into dest.
// if entries is not nil,  only the file or dir in entries will decompress.
func DecompressReader(name string, r io.ReadSeeker, dest string, entries ...string) error {
	size, err := getSize(r)
	if err != nil {
		return err
	}

	ar, err := NewReader(name, r, getReaderAt(r), size)
	if err != nil {
		return err
	}
	defer ar.Close()
	return ar.ExtractTo(dest, entries...)
}

// Compress creates a archive file in the location dest containing
// the contents of files listed in src. File paths
// can be those of regular files or directories. Regular
// files are stored at the 'root' of the archive, and
// directories are recursively added.
//
// Files with an extension for formats that are already
// compressed will be stored only, not compressed.
func Compress(dest string, src ...string) error {
	destf, err := os.OpenFile(dest, os.O_RDWR|os.O_TRUNC|os.O_CREATE, FileMode)
	if err != nil {
		return err
	}
	defer destf.Close()

	return CompressToWriter(dest, destf, src...)
}

// Compress the contents of files listed in src to w.
// File paths can be those of regular files or directories.
// Regula files are stored at the 'root' of the archive, and
// directories are recursively added.
//
// Files with an extension for formats that are already
// compressed will be stored only, not compressed.
func CompressToWriter(name string, w io.Writer, src ...string) error {
	aw, err := NewWriter(name, w)
	if err != nil {
		return err
	}
	defer aw.Close()

	size := int64(0)
	num := 0
	for _, path := range src {
		err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if DirMaxSize > 0 {
				size += info.Size()
				if size > DirMaxSize {
					return errors.New("Surpassed maximum archive size")
				}
			}
			if DirMaxFiles >= 0 {
				num++
				if num == DirMaxFiles+1 {
					return errors.New("Surpassed maximum number of files in archive")
				}
			}

			// must use filepath.Join, not "+"
			rel, _ := filepath.Rel(path, p)
			rel = filepath.Join(filepath.Base(path), rel)
			if info.Mode().IsDir() {
				return aw.AddEmptyDir(rel, info.Mode())
			}

			f, err := os.Open(p)
			if err != nil {
				return err
			}
			defer f.Close()

			return aw.AddReader(info, f, rel)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
