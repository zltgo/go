/*************************************************************************\
	功能 ：conf包封装了配置文件的相关操作，引入该包即可直接使用配置文件中的
		各类参数
\*************************************************************************/
package global

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/zltgo/fileserver/orm"
	. "github.com/zltgo/fileserver/utils"

	_ "github.com/mattn/go-sqlite3" //前面加'_'表示引入包（调用包的init），但是不引入包中的变量，函数等资源
)

type Config struct {
	HttpAddr        string              //服务地址
	DriverName      string              //数据库类型名
	DataSourceName  string              //数据库连接字符串
	Departments     []string            //部门名称，默认只有“开发部”
	RootPath        string              // 文件服务的根目录
	MaxUploadSize   int64               // 文件上传的最大字节数
	MaxDownloadSize int64               //打包下载的最大字节数
	ExtTable        string              //允许的文件后缀名列表
	ArchiveTable    string              //允许的压缩文件后缀名列表
	GroupAuthority  map[string][]string //分组权限表
}

var (
	AppPath       string
	AppConfigPath string
	Conf          Config       //配置信息结构
	Ds            *orm.DataSet //数据库操作对象
	GroupPattern  map[string][]*regexp.Regexp
)

//init函数会在包引入时自动调用，初始化只会执行一次
func init() {
	AppPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	AppConfigPath = filepath.Join(AppPath, "global", "app.conf")

	//默认值初始化
	Conf.HttpAddr = ":8080"
	Conf.DriverName = "sqlite3"
	Conf.DataSourceName = "./ftp.sqlite3"
	Conf.GroupAuthority = make(map[string][]string)
	Conf.Departments = append(Conf.Departments, "开发部")
	Conf.RootPath = "./fileDir"
	Conf.MaxUploadSize = 5 * 1024 * 1024 * 1024 //5GB
	Conf.MaxDownloadSize = 2 * Conf.MaxUploadSize

	//定义允许上传的文件扩展名
	Conf.ExtTable = ".dat,.gif,.jpg,.jpeg,.png,.bmp,.swf,.flv,.swf,.flv,.mp3,.wav,.wma,.wmv,.mid,.avi,.mpg,.asf,.rm,.rmvb,.doc,.docx,.xls,.xlsx,.ppt,.htm,.html,.txt,.zip,.tar,.rar,.gz,.bz2,.7z,.pdf,.chm"
	Conf.ArchiveTable = ".zip,.tar,.rar,.gz,.bz2"
	Conf.GroupAuthority["系统管理员"] = make([]string, 10)
	Conf.GroupAuthority["系统管理员"][0] = "^(GET|POST|PUT):/api/usr($|/)"
	Conf.GroupAuthority["系统管理员"][1] = "^(GET|POST|PUT|DELETE):/api/usrs($|/)"
	Conf.GroupAuthority["系统管理员"][2] = "^(GET|PUT):/api/conf$"
	Conf.GroupAuthority["系统管理员"][3] = "^(GET|POST|PUT|DELETE):/api/dir/"
	Conf.GroupAuthority["系统管理员"][4] = "^(GET|POST|PUT|DELETE):/api/file/"
	Conf.GroupAuthority["系统管理员"][5] = "^GET:/api/downloads($|/)"
	Conf.GroupAuthority["系统管理员"][6] = "^GET:/api/cnt($|/)"
	Conf.GroupAuthority["系统管理员"][7] = "^GET:/api/files($|/)"
	Conf.GroupAuthority["系统管理员"][8] = "^(GET|POST):/api/archive/"
	Conf.GroupAuthority["系统管理员"][9] = "^GET:/api/graph($|/)"

	Conf.GroupAuthority["配置管理员"] = make([]string, 10)
	Conf.GroupAuthority["配置管理员"][0] = "^(GET|POST|PUT):/api/usr($|/)"
	Conf.GroupAuthority["配置管理员"][1] = "^GET:/api/usrs($|/)"
	Conf.GroupAuthority["配置管理员"][2] = "^(GET|POST|PUT):/api/dir/"
	Conf.GroupAuthority["配置管理员"][3] = "^(GET|POST|PUT|DELETE):/api/file/"
	Conf.GroupAuthority["配置管理员"][4] = "^GET:/api/downloads($|/)"
	Conf.GroupAuthority["配置管理员"][5] = "^GET:/api/cnt($|/)"
	Conf.GroupAuthority["配置管理员"][6] = "^GET:/api/files($|/)"
	Conf.GroupAuthority["配置管理员"][7] = "^(GET|POST):/api/archive/"
	Conf.GroupAuthority["配置管理员"][8] = "^GET:/api/graph($|/)"
	Conf.GroupAuthority["配置管理员"][9] = "^(GET):/api/conf$"

	Conf.GroupAuthority["普通用户"] = make([]string, 7)
	Conf.GroupAuthority["普通用户"][0] = "^(GET|POST|PUT):/api/usr($|/)"
	Conf.GroupAuthority["普通用户"][1] = "^GET:/api/dir/"
	Conf.GroupAuthority["普通用户"][2] = "^GET:/api/file/"
	Conf.GroupAuthority["普通用户"][3] = "^GET:/api/cnt($|/)"
	Conf.GroupAuthority["普通用户"][4] = "^GET:/api/files($|/)"
	Conf.GroupAuthority["普通用户"][5] = "^GET:/api/archive/"
	Conf.GroupAuthority["普通用户"][6] = "^GET:/api/graph($|/)"

	Conf.GroupAuthority["游客"] = make([]string, 5)
	Conf.GroupAuthority["游客"][0] = "^(GET|POST|PUT):/api/usr($|/)"
	Conf.GroupAuthority["游客"][1] = "^GET:/api/dir/"
	Conf.GroupAuthority["游客"][2] = "^GET:/api/cnt($|/)"
	Conf.GroupAuthority["游客"][3] = "^GET:/api/files($|/)"
	Conf.GroupAuthority["游客"][4] = "^GET:/api/graph($|/)"

	//打开配置文件，没有则创建，权限参数待改进
	file, err := os.OpenFile(AppConfigPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	CheckErr(err)
	defer file.Close()

	//读取配置文件，JSON格式，没有则写入默认值
	buff, _ := ioutil.ReadAll(file)
	if len(buff) == 0 {
		buff, err = json.MarshalIndent(Conf, "", "	")
		CheckErr(err)
		_, err = file.Write(buff)
	} else {
		err = json.Unmarshal(buff, &Conf)
	}
	CheckErr(err)

	//初始化数据库操作对象DS
	Ds, err = orm.NewDataSet(Conf.DriverName, Conf.DataSourceName)
	CheckErr(err)

	//权限控制
	GroupPattern = make(map[string][]*regexp.Regexp, len(Conf.GroupAuthority))
	for k, urls := range Conf.GroupAuthority {
		GroupPattern[k] = make([]*regexp.Regexp, len(urls))
		for i, url := range urls {
			GroupPattern[k][i] = regexp.MustCompile(url)
		}
	}
	return
}
