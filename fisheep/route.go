package main

import (
	"encoding/json"
	"errors"
	. "fisheep/conf"
	m "fisheep/model"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zltgo/api"
	"github.com/zltgo/api/jwt"
	"github.com/zltgo/api/session"
	"github.com/zltgo/structure"
	"gopkg.in/mgo.v2"
)

type testData struct {
	Code int    `default:"200"`
	MIME string `default:"application/json"`
}

// Wrap a test middleware.
func TestMware(ctx *gin.Context) {
	//open configuration file
	testPath := filepath.Join("./test", ctx.Request.URL.Path)
	file, err := os.Open(testPath)
	if err != nil {
		ctx.Next()
		return
	}
	defer file.Close()

	var rv testData
	if err = structure.SetDefault(&rv); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	// read data
	buff, _ := ioutil.ReadAll(file)
	if err = json.Unmarshal(buff, &rv); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.Header("Content-Type", rv.MIME)
	ctx.Status(rv.Code)
	ctx.Writer.Write(buff)
	return
}

func publicRoutes() *gin.Engine {
	g := gin.Default()
	g.Static("/static", "./static")

	// 测试时在conf.go中配置debug模式
	// 如果请求的url在test目录下存在，则返回test目录下配置的json数据
	if gin.Mode() == "test" {
		g.Use(TestMware)
	}

	// Make jwt.AuthMware
	auth := jwt.NewAuthByOpts(Pub.AuthOpts)

	// Use default cookie provider
	session.DefaultCookiePd.SetCookieRatelimit(Pub.CookieRatelimit)
	session.DefaultCookiePd.SetUrlRatelimit(Pub.UrlRatelimit)
	// api ratelimits
	ratelimiter := session.DefaultCookiePd.SessionMware

	// Make mongo middleware
	mongo := func(next api.Handler) api.Handler {
		return func(ctx *api.Context) {
			ms := m.MS.Clone()
			defer ms.Close()
			ctx.Map(ms.DB(""))
			next(ctx)
		}
	}

	mw := api.NewChain(ratelimiter, auth.AuthMware, mongo)

	var rs api.Routes
	//基础接口
	rs.Add("GET:/", m.Index)
	rs.Add("GET:/token", ratelimiter, auth.RefreshHandler)
	if gin.Mode() == gin.TestMode {
		rs.Add("GET:/sms", ratelimiter, mongo, m.TestSms)
		rs.Add("GET:/captcha", ratelimiter, m.TestCaptcha)
	} else {
		rs.Add("GET:/captcha", ratelimiter, m.GetCaptcha)
	}

	//用户接口
	rs.Add("POST:/login/usr", ratelimiter, mongo, auth.LoginHandler(m.UsrLogin))
	rs.Add("PUT:/usr/pwd", mw.Then(auth.LoginHandler(m.ResetPwd)))
	rs.Add("PUT:/usr/phone", mw.Then(auth.LoginHandler(m.ResetPhone)))
	rs.Add("PUT:/usr/addrs", mw.Then(m.SetAddrs))

	// 浏览商品
	rs.Add("GET:/goods", mw.Then(m.ViewGoods))

	rs.RegisterToGin(g)
	return g
}

func privateRoutes() *gin.Engine {
	g := gin.Default()
	g.Static("/static", "./static")

	// 测试时在conf.go中配置debug模式
	// 如果请求的url在test目录下存在，则返回test目录下配置的json数据
	if gin.Mode() == "test" {
		g.Use(TestMware)
	}

	// Make jwt.AuthMware
	auth := jwt.NewAuthByOpts(Pri.AuthOpts)

	// Use default cookie provider
	session.DefaultCookiePd.SetCookieRatelimit(Pri.CookieRatelimit)
	session.DefaultCookiePd.SetUrlRatelimit(Pri.UrlRatelimit)
	// api ratelimits
	ratelimiter := session.DefaultCookiePd.SessionMware

	// Make mongo middleware
	mongo := func(next api.Handler) api.Handler {
		return func(ctx *api.Context) {
			ms := m.MS.Clone()
			defer ms.Close()
			ctx.Map(ms.DB(""))
			next(ctx)
		}
	}

	mw := api.NewChain(ratelimiter, auth.AuthMware, mongo, Authorizator)

	var rs api.Routes
	//基础接口
	rs.Add("GET:/", m.Manage)
	rs.Add("GET:/token", ratelimiter, auth.RefreshHandler)
	if gin.Mode() == gin.TestMode {
		rs.Add("GET:/captcha", ratelimiter, m.TestCaptcha)
	} else {
		rs.Add("GET:/captcha", ratelimiter, m.GetCaptcha)
	}

	//管理员用户接口
	rs.Add("POST:/login/manager", ratelimiter, mongo, auth.LoginHandler(m.ManagerLogin))
	rs.Add("POST:/manager", mw.Then(m.AddManager))
	rs.Add("PUT:/manager", mw.Then(m.ModifyManager))
	rs.Add("DELETE:/manager/:id", mw.Then(m.RemoveManager))
	rs.Add("GET:/managers", mw.Then(m.GetManagers))

	// 编辑Areas接口
	rs.Add("POST:/area", mw.Then(m.Nodes(m.Areas).Add))
	rs.Add("PUT:/area/name", mw.Then(m.Nodes(m.Areas).ResetName))
	rs.Add("PUT:/area/pid", mw.Then(m.Nodes(m.Areas).ResetPid))
	rs.Add("GET:/areas", mw.Then(m.Nodes(m.Areas).GetChildren))
	rs.Add("DELETE:/areas", mw.Then(m.Nodes(m.Areas).Remove))
	rs.Add("PUT:/areas", mw.Then(m.Nodes(m.Areas).RebuildAll))

	// 编辑Products接口
	rs.Add("POST:/product", mw.Then(m.Nodes(m.Products).Add))
	rs.Add("PUT:/product/name", mw.Then(m.Nodes(m.Products).ResetName))
	rs.Add("PUT:/product/pid", mw.Then(m.Nodes(m.Products).ResetPid))
	rs.Add("GET:/products", mw.Then(m.Nodes(m.Products).GetChildren))
	rs.Add("DELETE:/products", mw.Then(m.Nodes(m.Products).Remove))
	rs.Add("PUT:/products", mw.Then(m.Nodes(m.Products).RebuildAll))

	// 商品管理
	rs.Add("POST:/goods", mw.Then(m.AddGoods))
	rs.Add("GET:/goods", mw.Then(m.GetGoods))
	rs.Add("PUT:/goods", mw.Then(m.ModifyGoods))
	rs.Add("DELETE:/goods/:id", mw.Then(m.RemoveGoods))
	rs.Add("POST:/image/:id", mw.Then(m.UploadImage))

	rs.RegisterToGin(g)
	return g
}

// Make Authority Limit
func Authorizator(next api.Handler) api.Handler {
	return func(ctx *api.Context) {
		var uid jwt.UID
		ctx.MustGet(&uid)

		var db *mgo.Database
		ctx.MustGet(&db)

		mb, code, err := m.GetManagerDb(db, uid)
		if err != nil {
			ctx.Render(code, err)
			return
		}
		ctx.Map(mb)

		// 检查接口调用权限，超级用户拥有所有权限
		if mb.Grp == m.SuperGrp || checkAuthority(ctx.Request, mb.Grp, Pri.AuthorityLimit) {
			next(ctx)
		} else {
			ctx.Render(http.StatusForbidden, errors.New(mb.Grp+mb.Usr+"无权访问"))
		}

		return
	}
}

// 检查访问权限
func checkAuthority(r *http.Request, grp string, mp map[string]string) bool {
	// find url key and ratelimit configuration in mp.
	// ensure evey url has a ratelimit configuration.
	url1 := r.Method + ":" + r.URL.Path + "/"
	url2 := "ANY:" + r.URL.Path + "/"

	grp = "," + grp + "," //防止A到"AB,CD"
	for k, grps := range mp {
		if strings.HasPrefix(url1, k) || strings.HasPrefix(url2, k) {
			if strings.Contains(grps, grp) {
				return true
			}
		}
	}
	//未配置的均不能通过
	return false
}
