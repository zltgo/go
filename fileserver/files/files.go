package files

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zltgo/archive"
)

const UpdateTime = 1 * time.Minute

type FileInfo struct {
	Path     string
	Name     string
	IsDir    bool
	FileSize int64
	ModTime  int64 //修改时间，单位为秒
}

//在目录中检索文档
type RootFiles struct {
	rootPath string
	fileList []FileInfo //[]FileInfo为nil说明rootPath不正确
	fileMap  map[string]FileInfo
	mutex    sync.RWMutex
}

//创建检索文档类
func NewRootFiles(root string) (*RootFiles, error) {
	fi, err := os.Stat(root)
	if err != nil || !fi.IsDir() {
		return nil, &os.PathError{"NewRootFiles", root, errors.New("not a directory")}
	}

	rfs := &RootFiles{rootPath: root}
	go rfs.Update()
	return rfs, nil
}

//更新指定目录下的文件信息
func (m *RootFiles) Update() {
	var tmp []FileInfo
	m.CountAll("/", &tmp)
	mp := make(map[string]FileInfo, len(tmp))

	for i := 0; i < len(tmp); i++ {
		mp[filepath.Join(tmp[i].Path, tmp[i].Name)] = tmp[i]
	}
	m.mutex.Lock()
	m.fileList = tmp
	m.fileMap = mp
	m.mutex.Unlock()

	//定时更新
	time.AfterFunc(UpdateTime, m.Update)
}

//统计指定目录（含子目录）下所有的文件和目录数量
func (m *RootFiles) CountAll(path string, fileList *[]FileInfo) (int64, error) {
	files, err := ioutil.ReadDir(filepath.Join(m.rootPath, path))
	if err != nil {
		return 0, err
	}
	//循环计数,统计文件大小
	var sum int64
	for _, fi := range files {
		size := fi.Size()
		if fi.IsDir() {
			size, _ = m.CountAll(filepath.Join(path, fi.Name()), fileList)
		}
		sum += size
		*fileList = append(*fileList, FileInfo{path, fi.Name(), fi.IsDir(), size, fi.ModTime().Unix()})
	}
	return sum, nil
}

//列出指定目录下的文件信息，rv为nil说明路径不正确
func (m *RootFiles) ViewPath(path string) ([]FileInfo, error) {
	files, err := ioutil.ReadDir(filepath.Join(m.rootPath, path))
	if err != nil {
		return nil, err
	}
	//循环计数,并查找文件夹的大小信息
	fileList := make([]FileInfo, len(files))

	m.mutex.RLock()
	for i, fi := range files {
		size := fi.Size()
		if fi.IsDir() {
			if tmp, ok := m.fileMap[filepath.Join(path, fi.Name())]; ok {
				size = tmp.FileSize
			}
		}
		fileList[i] = FileInfo{path, fi.Name(), fi.IsDir(), size, fi.ModTime().Unix()}
	}
	m.mutex.RUnlock()

	return fileList, nil
}

//按正则表达式搜索指定目录下的文件信息，rv为nil说明路径不正确
func (m *RootFiles) SearchPath(path string, rx *regexp.Regexp) (rv []FileInfo, err error) {
	//判断路径合法性
	_, err = m.Stat(path)
	if err != nil {
		return
	}

	rv = make([]FileInfo, 0)
	m.mutex.RLock()
	for _, fi := range m.fileList {
		if strings.HasPrefix(fi.Path, path) && (rx == nil || rx.MatchString(fi.Name)) {
			rv = append(rv, fi)
		}
	}
	m.mutex.RUnlock()
	return
}

//分割路径，返回上级目录和本级名称
func Split(path string) (dir, name string) {
	if path == "" || path[0] != '/' {
		path = "/" + path
	}
	_, name = filepath.Split(path)
	dir = filepath.Dir(path)
	return
}

//更改文件或目录名
func (m *RootFiles) Rename(oldpath, newpath string) error {
	return os.Rename(filepath.Join(m.rootPath, oldpath), filepath.Join(m.rootPath, newpath))
}

//删除一个文件或空文件夹
func (m *RootFiles) Remove(path string) error {
	return os.Remove(filepath.Join(m.rootPath, path))
}

//删除一个文件夹及其子文件
func (m *RootFiles) RemoveAll(path string) error {
	if path == "" || path == "/" {
		return errors.New("不能删除根目录")
	}
	return os.RemoveAll(filepath.Join(m.rootPath, path))
}

//创建一个文件夹
func (m *RootFiles) AddDir(path string) error {
	return os.Mkdir(filepath.Join(m.rootPath, path), os.ModePerm)
}

//判断文件或目录是否存在
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

//判断文件或目录是否存在，相对路径
func (m *RootFiles) IsExist(path string) bool {
	_, err := os.Stat(filepath.Join(m.rootPath, path))
	return err == nil || os.IsExist(err)
}

//重名时自动重命名
// a ->  a(1)
// a.b -> a(1).b
// .a -> .a(1)
// a. -> a.(1)
// a.b. -> a(1).b.
// .b.c.d -> .b.c.d(1)
func UniquePath(fpath string) string {
	if !IsExist(fpath) {
		return fpath
	}

	dir, name := filepath.Split(fpath)
	prefix := name
	ext := ""
	if index := strings.Index(name, "."); index > 0 && index < len(name)-1 {
		prefix = name[:index]
		ext = name[index:]
	}

	for n := 1; true; n++ {
		fpath = dir + prefix + fmt.Sprintf("(%d)", n) + ext
		if !IsExist(fpath) {
			break
		}
	}
	return fpath
}

//获取文件状态
func (m *RootFiles) Stat(path string) (*FileInfo, error) {
	fi, err := os.Stat(filepath.Join(m.rootPath, path))
	if err != nil {
		return nil, err
	}

	size := fi.Size()
	if fi.IsDir() {
		m.mutex.RLock()
		if tmp, ok := m.fileMap[path]; ok {
			size = tmp.FileSize
		}
		m.mutex.RUnlock()
	}

	return &FileInfo{path, fi.Name(), fi.IsDir(), size, fi.ModTime().Unix()}, nil
}

//创建一个文件，重名时自动重命名
func (m *RootFiles) AddFile(path string, src io.Reader) error {
	path = UniquePath(filepath.Join(m.rootPath, path))
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_EXCL|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, src)
	f.Close()
	return err
}

//提供下载
func (m *RootFiles) Download(w http.ResponseWriter, r *http.Request, path string) error {
	f, err := os.Open(filepath.Join(m.rootPath, path))
	if err != nil {
		return err
	}

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	//必须再判断一下，目录也能成功open
	if fi.IsDir() {
		return &os.PathError{"download file:", path, errors.New("is a directory")}
	}
	//设置http头为下载
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+fi.Name())

	//不能用这个接口，会将/api/file/index.html重定向到/api/file/
	//http.ServeFile(w, r, filepath.Join(m.rootPath, path))

	http.ServeContent(w, r, path, fi.ModTime(), f)
	return nil
}

//提供压缩并下载
func (m *RootFiles) GetArchive(w http.ResponseWriter, r *http.Request, path string, ext string, maxSize int64) error {
	m.mutex.RLock()
	fi, ok := m.fileMap[path]
	m.mutex.RUnlock()

	if !ok {
		return &os.PathError{"GetArchive", path, os.ErrNotExist}
	}

	if fi.FileSize > maxSize {
		return errors.New("file size too large:" + path)
	}

	//	tmp, err := ioutil.TempDir("", "GetArchive")
	//	if err != nil {
	//		return err
	//	}
	//	defer os.RemoveAll(tmp)

	archiveName := fi.Name + ext
	//	tmpDir := filepath.Join(tmp, archiveName)
	//	if err := archive.Compress(tmpDir, filepath.Join(m.rootPath, path)); err != nil {
	//		return err
	//	}

	//设置http头为下载
	w.Header().Set("Last-Modified", time.Unix(fi.ModTime, 0).UTC().Format(http.TimeFormat))
	w.Header().Set("Content-Type", "application/zip; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename="+archiveName)

	//下载文件
	//http.ServeFile(w, r, tmpDir)
	archive.CompressToWriter(archiveName, w, filepath.Join(m.rootPath, path))
	return nil
}

//添加一个压缩文件并解压到指定目录
func (m *RootFiles) AddArchive(name string, r multipart.File, path string) error {
	return archive.DecompressReader(name, r, filepath.Join(m.rootPath, path))
}
