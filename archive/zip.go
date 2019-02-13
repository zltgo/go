package archive

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

func init() {
	RegisterWriter(".zip", NewZipWriter)
	RegisterReader(".zip", NewZipReader)
}

type zipWriter struct {
	zw *zip.Writer
}

type zipReader struct {
	zr *zip.Reader
}

func NewZipWriter(w io.Writer) (Writer, error) {
	return zipWriter{zip.NewWriter(w)}, nil
}

func NewZipReader(r io.Reader, ra io.ReaderAt, size int64) (Reader, error) {
	zr, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, err
	}
	return zipReader{zr}, nil
}
func (m zipWriter) AddEmptyDir(relPath string, perm os.FileMode) error {
	if !strings.HasSuffix(relPath, "/") {
		relPath += "/"
	}
	h := &zip.FileHeader{
		Name:   relPath,
		Method: zip.Store,
	}
	h.SetMode(perm)
	h.SetModTime(time.Now())
	_, err := m.zw.CreateHeader(h)
	return err
}

func (m zipWriter) AddReader(info os.FileInfo, r io.Reader, relPath string) error {
	if !info.Mode().IsRegular() {
		return errors.New("Only regular files supported: " + info.Name())
	}

	h, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	if relPath == "" || strings.HasSuffix(relPath, "/") {
		relPath += info.Name()
	}
	h.Name = relPath
	ext := strings.ToLower(path.Ext(h.Name))
	if CompressedFormats[ext] {
		h.Method = zip.Store
	} else {
		h.Method = zip.Deflate
	}

	ww, err := m.zw.CreateHeader(h)
	if err != nil {
		return err
	}

	_, err = io.Copy(ww, r)
	return err
}

func (m zipWriter) Close() error {
	return m.zw.Close()
}

func (m zipReader) Close() error {
	return nil
}

func (m zipReader) ExtractTo(dest string, entries ...string) error {
	var dirMap = make(map[string]string)
	for _, zf := range m.zr.File {
		if !isEntry(zf.Name, entries) {
			continue
		}

		//if fpath already exist, chang the fpath to dir(n)/file
		if err := unzipFile(zf, dest, dirMap); err != nil {
			return err
		}
	}

	return nil
}

// decompress a zip file or directory
func unzipFile(zf *zip.File, dest string, dirMap map[string]string) error {
	var err error
	//if fpath already exist, change the fpath to dir(n)/file
	for {
		fpath := generatePath(dirMap, dest, zf.Name)

		if zf.FileInfo().IsDir() {
			err = os.MkdirAll(fpath, os.ModePerm)
		} else {
			rc, err := zf.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			err = writeNewFile(fpath, rc, zf.FileInfo().Mode())
		}

		if !os.IsExist(err) {
			break
		}
	}

	return err
}
