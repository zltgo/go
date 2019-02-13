package archive

import (
	"io"
	"os"

	"github.com/nwaples/rardecode"
)

//for rar decode
func init() {
	RegisterReader(".rar", NewRarReader)
}

type rarReader struct {
	rr *rardecode.Reader
}

func NewRarReader(r io.Reader, ra io.ReaderAt, size int64) (Reader, error) {
	rr, err := rardecode.NewReader(r, "")
	if err != nil {
		return nil, err
	}
	return rarReader{rr}, nil
}

func (m rarReader) Close() error {
	return nil
}

func (m rarReader) ExtractTo(dest string, entries ...string) error {
	var dirMap = make(map[string]string)
	for {
		header, err := m.rr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if !isEntry(header.Name, entries) {
			continue
		}

		//if fpath already exist, chang the fpath to dir(n)/file
		if err := unrarFile(m.rr, header, dest, dirMap); err != nil {
			return err
		}
	}

	return nil
}

// decompress a rar file or directory
func unrarFile(rr *rardecode.Reader, header *rardecode.FileHeader, dest string, dirMap map[string]string) error {
	var err error
	//if fpath already exist, change the fpath to dir(n)/file
	for {
		fpath := generatePath(dirMap, dest, header.Name)

		if header.IsDir {
			err = os.MkdirAll(fpath, os.ModePerm)
		} else {
			err = writeNewFile(fpath, rr, header.Mode())
		}

		if !os.IsExist(err) {
			break
		}
	}

	return err
}
