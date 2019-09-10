package conf

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/zltgo/api/bind"
	"github.com/zltgo/api/jwt"
	"github.com/zltgo/structure"
)

var (
	confDir = flag.String("f", "./conf", "Directory of configuration file")

	Pub     = PubConf{}
	Pri     = PriConf{}
	ConfDir = "./conf"
)

//system configuration
type PubConf struct {
	HttpAddr string `default:":8080"`

	// Used for jwt.NewAuth
	AuthOpts jwt.AuthOpts

	// rate limit config for create cookie
	CookieRatelimit []int `default:"20,60,100,86400"`

	// rate limit config for url per client, see"github.com/zltgo/api/session"
	UrlRatelimit map[string][]int

	// Url for Dial with mongodb, it must include database name.
	MongoUrl string `default:"mongodb://localhost:27017/fisheep"`

	// Milliseconds to wait for Write mongodb before timing out.
	WTimeout int `default:"2000"`

	// Sms有效时间，默认为10分钟
	SmsMaxAge      int `default:"600"`
	SmsMinAge      int `default:"60"`                       // 单个手机获取短信验证码的最小间隔（秒）
	SmsMaxCntDaily int `default:"10"`                       // 单个手机每天获取短信验证码的最大次数
	SmsCodeNum     int `default:"4" validate:"min=4,max=6"` //短信验证码位数

	// Used for captcha
	CaptchaWide    int `default:"112"`
	CaptchaHeight  int `default:"64"`
	CaptchaDisturb int `default:"8" validate:"min=4,max=32"`
	CaptchaNum     int `default:"4" validate:"min=4,max=6"` //图形验证码个数
}

type PriConf struct {
	GinMode  string `default:"test" validate:"eq=debug|eq=release|eq=test"`
	HttpAddr string `default:":8888"`
	// Used for jwt.NewAuth
	AuthOpts jwt.AuthOpts

	// rate limit config for create cookie
	CookieRatelimit []int `default:"20,60,100,86400"`

	// rate limit config for url per client, see"github.com/zltgo/api/session"
	UrlRatelimit map[string][]int

	// limits of authority, for example:
	// map{"ANY:/usr/" : ",系统管理员,普通管理员,"}
	// 说明：url要以"/"结尾，防止"/usr"和"/usrs"错判
	// 拥有访问权限的用户组用","分开，开头和结尾必须都有。
	AuthorityLimit map[string]string

	// Url for Dial with mongodb, it must include database name.
	MongoUrl string `default:"mongodb://localhost:27017/fisheep"`

	// Milliseconds to wait for Write mongodb before timing out.
	WTimeout int `default:"2000"`

	// Used for captcha
	CaptchaWide    int `default:"112"`
	CaptchaHeight  int `default:"64"`
	CaptchaDisturb int `default:"8" validate:"min=4,max=32"`
	CaptchaNum     int `default:"4" validate:"min=4,max=6"` //图形验证码个数

	// 上传图片的最大值，默认最大1M
	MaxUploadSize int64 `default:"1048576"`
	//允许上传的文件后缀名列表，必须以逗号开头和结尾
	ExtTable string `default:",.gif,.jpg,.jpeg,.png,.bmp,.html,.txt,.pdf,"`
}

func init() {
	// parse commandline flags
	flag.Parse()
	ConfDir = *confDir
	log.Println("Directory of configuration file is:", ConfDir)

	if err := InitConf(filepath.Join(ConfDir, "pubconf.json"), &Pub); err != nil {
		log.Panicln("init pubconf.json:", err)
	}
	if err := InitConf(filepath.Join(ConfDir, "priconf.json"), &Pri); err != nil {
		log.Panicln("init priconf.json:", err)
	}
	return
}

func InitConf(confPath string, ptr interface{}) error {
	file, err := os.OpenFile(confPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	// read conf or write default values.
	buff, _ := ioutil.ReadAll(file)
	if len(buff) == 0 {
		err = structure.SetDefault(ptr)
		if err != nil {
			return errors.New("SetDefault: " + err.Error())
		}

		buff, err = json.MarshalIndent(ptr, "", "	")
		if err != nil {
			return errors.New("json.MarshalIndent: " + err.Error())
		}

		if _, err = file.Write(buff); err != nil {
			return errors.New("file.Write: " + err.Error())
		}
	} else {
		if err = json.Unmarshal(buff, ptr); err != nil {
			return errors.New("json.Unmarshal: " + err.Error())
		}

		//validate
		if err = bind.DefaultValidator.Struct(ptr); err != nil {
			return errors.New("validate: " + err.Error())
		}
	}

	return nil
}
