package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dsnet/compress/bzip2"
)

//for tar/tar.gz/tar.bz2
func init() {
	RegisterWriter(".tar", NewTarWriter)
	RegisterReader(".tar", NewTarReader)
	RegisterWriter(".gz", NewTarGzWriter)
	RegisterReader(".gz", NewTarGzReader)
	RegisterWriter(".bz2", NewTarBz2Writer)
	RegisterReader(".bz2", NewTarBz2Reader)
}

type tarWriter struct {
	tw *tar.Writer
	c  io.Closer
}

type tarReader struct {
	tr *tar.Reader
	c  io.Closer
}

func NewTarWriter(w io.Writer) (Writer, error) {
	return tarWriter{tar.NewWriter(w), nil}, nil
}

func NewTarReader(r io.Reader, ra io.ReaderAt, size int64) (Reader, error) {
	return tarReader{tar.NewReader(r), nil}, nil
}

func NewTarGzWriter(w io.Writer) (Writer, error) {
	gw := gzip.NewWriter(w)
	return tarWriter{tar.NewWriter(gw), gw}, nil
}

func NewTarGzReader(r io.Reader, ra io.ReaderAt, size int64) (Reader, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return tarReader{tar.NewReader(gr), gr}, nil
}

func NewTarBz2Writer(w io.Writer) (Writer, error) {
	bz2w, err := bzip2.NewWriter(w, nil)
	if err != nil {
		return nil, err
	}
	return tarWriter{tar.NewWriter(bz2w), bz2w}, nil
}

func NewTarBz2Reader(r io.Reader, ra io.ReaderAt, size int64) (Reader, error) {
	bz2r, err := bzip2.NewReader(r, nil)
	if err != nil {
		return nil, err
	}
	return tarReader{tar.NewReader(bz2r), bz2r}, nil
}

func (m tarWriter) Close() error {
	err := m.tw.Close()
	if m.c != nil {
		return m.c.Close()
	}
	return err
}

func (m tarReader) Close() error {
	if m.c != nil {
		return m.c.Close()
	}
	return nil
}

func (m tarWriter) AddEmptyDir(relPath string, perm os.FileMode) error {
	tmp, err := ioutil.TempDir("", "AddEmptyDir")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	dir := filepath.Join(tmp, relPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	h, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	h.Name = relPath
	return m.tw.WriteHeader(h)
}

func (m tarWriter) AddReader(info os.FileInfo, r io.Reader, relPath string) error {
	h, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	if relPath == "" || strings.HasSuffix(relPath, "/") {
		relPath += info.Name()
	}
	h.Name = relPath

	if err := m.tw.WriteHeader(h); err != nil {
		return err
	}

	_, err = io.Copy(m.tw, r)
	return err
}

func (m tarReader) ExtractTo(dest string, entries ...string) error {
	var dirMap = make(map[string]string)
	for {
		header, err := m.tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !isEntry(header.Name, entries) {
			continue
		}

		//if fpath already exist, chang the fpath to dir(n)/file
		if err := untarFile(m.tr, header, dest, dirMap); err != nil {
			return err
		}
	}
	return nil

}

// untarFile untars a single file from tr with header header into destination.
func untarFile(tr *tar.Reader, header *tar.Header, dest string, dirMap map[string]string) error {
	var err error
	//if fpath already exist, change the fpath to dir(n)/file
	for {
		fpath := generatePath(dirMap, dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(fpath, os.ModePerm)
		case tar.TypeReg, tar.TypeRegA:
			err = writeNewFile(fpath, tr, header.FileInfo().Mode())
		case tar.TypeSymlink:
			err = writeNewSymbolicLink(fpath, header.Linkname)
		default:
			err = fmt.Errorf("%s: unknown type flag: %c", header.Name, header.Typeflag)
		}

		if !os.IsExist(err) {
			break
		}
	}
	return err
}
