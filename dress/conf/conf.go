package conf

import (
	"github.com/zltgo/dress/model"
	"github.com/zltgo/reflectx"
	"github.com/zltgo/webkit/jwt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

type Conf struct {
	ServeAddr string `default:":8080"`
	Model        model.Options
	AuthOpts  jwt.AuthOpts
	Graphql   struct {
		CacheSize       int `default:"1000"` //缓存限制，缓存的是graphql语句的解析结果
		ComplexityLimit int `default:"1000"` //复杂度限制，计算递归查询的总计数
	}
}

func ReadCfg(path string) *Conf {
	//set default values by tag
	cfg := Conf{}
	reflectx.SetDefault(&cfg)
	setDefaultRateLimit(&cfg.Model)


	//打开配置文件，没有则创建，权限参数待改进
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	checkErr(err, "os.OpenFile")
	defer file.Close()

	//读取配置文件，yaml格式，没有则写入默认值
	buff, _ := ioutil.ReadAll(file)
	if len(buff) == 0 {
		buff, err = yaml.Marshal(&cfg)
		checkErr(err, "yaml.Marshal")

		_, err = file.Write(buff)
		checkErr(err, "write yaml")
	} else {
		checkErr(yaml.Unmarshal(buff, &cfg), "yaml.Unmarshal")
	}
	return &cfg
}

//exit if any error occurred.
func checkErr(err error, prefix string) {
	if err != nil {
		log.Fatalf("%s error: %v", prefix, err)
	}
	return
}

func setDefaultRateLimit(opts *model.Options) {
	opts.IPRates = make(map[string][]int)
	opts.IPRates["GetAccountIdByMobile"] = []int{100, 86400}
	opts.IPRates["Admin"] = []int{10, 86400}
	opts.IPRates["Login"] = []int{100, 86400}
	opts.IPRates["RefreshToken"] = []int{100, 86400}
	opts.IPRates["Any"] = []int{10000,86400} //all functions

	opts.UserRates = make(map[string][]int)
	opts.UserRates["SearchAccounts"] = []int{100, 3600}
}