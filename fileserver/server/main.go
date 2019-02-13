package main

import (
	"github.com/zltgo/fileserver/server/global"

	m "github.com/zltgo/fileserver/server/models"

	"github.com/zltgo/api"
	"github.com/zltgo/api/cache"
	"github.com/zltgo/api/session"
)

func main() {
	a := api.Default()
	a.Static("/static", "static")
	a.AddRoutes(Routes())

	//g.RunTLS(global.Conf.HttpAddr, "./pem/cert.pem", "./pem/key.pem")
	a.Run(global.Conf.HttpAddr)
}

func Routes() api.Routes {
	rs := make(api.Routes, 0)
	lmc := cache.NewLruMemCache(1024)
	cs := session.NewCookieProvider(session.CookieOpts{}, lmc)
	se := cs.SessionHandler

	//用户操作
	rs.Add("GET:/", m.Index)
	rs.Add("POST:/api/login", se, m.Login)
	rs.Add("GET:/api/logout", se, m.Logout)

	rs.Add("GET:/api/captcha", se, m.Captcha)
	rs.Add("POST:/api/usr", se, m.Register)
	rs.Add("GET:/api/usr", se, m.AuthRequire, m.GetUsrInfo)
	rs.Add("PUT:/api/usr", se, m.AuthRequire, m.ResetPwd)

	//用户管理
	rs.Add("GET:/api/usrs", se, m.AuthRequire, m.GetUsrs)
	rs.Add("POST:/api/usrs", se, m.AuthRequire, m.AddUsr)
	rs.Add("PUT:/api/usrs", se, m.AuthRequire, m.ModifyUsr)
	rs.Add("DELETE:/api/usrs", se, m.AuthRequire, m.RemoveUsr)

	//配置文件管理
	rs.Add("GET:/api/conf", se, m.AuthRequire, m.GetConf)
	rs.Add("PUT:/api/conf", se, m.AuthRequire, m.SetConf)

	//文件管理
	rs.Add("GET:/api/file/*path", se, m.AuthRequire, m.DownloadFile)
	rs.Add("POST:/api/file/*path", se, m.AuthRequire, m.UploadFile)
	rs.Add("PUT:/api/file/*path", se, m.AuthRequire, m.RenameFile)
	rs.Add("DELETE:/api/file/*path", se, m.AuthRequire, m.RemoveFile)
	rs.Add("GET:/api/files/*path", se, m.AuthRequire, m.Search)

	rs.Add("GET:/api/dir/*path", se, m.AuthRequire, m.ViewPath)
	rs.Add("POST:/api/dir/*path", se, m.AuthRequire, m.AddDir)
	rs.Add("PUT:/api/dir/*path", se, m.AuthRequire, m.RenameDir)
	rs.Add("DELETE:/api/dir/*path", se, m.AuthRequire, m.RemoveDir)

	//下载统计与查询
	rs.Add("GET:/api/downloads/*path", se, m.AuthRequire, m.GetDownloads)
	rs.Add("GET:/api/cnt/*path", se, m.AuthRequire, m.GetCnt)
	rs.Add("GET:/api/graph/*path", se, m.AuthRequire, m.GetGraph)

	//打包上传下载
	rs.Add("GET:/api/archive/*path", se, m.AuthRequire, m.DownloadArchive)
	rs.Add("POST:/api/archive/*path", se, m.AuthRequire, m.UploadArchive)

	return rs
}
