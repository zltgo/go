package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/zltgo/fileserver/server/global"

	"github.com/zltgo/fileserver/files"
	. "github.com/zltgo/fileserver/utils"
)

const createFileTablesIfNotExists = `create table if not exists downloads(
	uid  int  NOT NULL,
	path varchar(255) NOT NULL,
	ip bigint NOT NULL,
	time bigint   NOT NULL,
	year int NOT NULL,
	month int NOT NULL,
	day int  NOT NULL,
	weekday int NOT NULL);`

type downDb struct {
	Uid     int64  `uid`
	Path    string `path`
	Ip      int64  `ip`
	Time    int64  `time`
	Year    int    `year`
	Month   int    `month`
	Day     int    `day`
	Weekday int    `weekday`
}

var (
	m_publicFiles *files.RootFiles
)

func init() {
	var err error
	m_publicFiles, err = files.NewRootFiles(global.Conf.RootPath)
	CheckErr(err)

	_, err = global.Ds.Exec(createFileTablesIfNotExists)
	CheckErr(err)

	//绑定结构体与表格
	global.Ds.Register("downloads", downDb{})
	global.Ds.Register("downloads inner join users on downloads.uid = users.rowid", downloadDb{})

	return
}

//Get：files/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//404：资源不存在
//500：服务器错误
func Search(r *http.Request) (interface{}, error) {
	var err error
	path := r.URL.Path[len("/api/files"):]
	var rx *regexp.Regexp
	expr := r.FormValue("Expr")
	if expr != "" {
		rx, err = regexp.Compile(expr)
		if err != nil {
			return 400, errors.New("搜索条件错误：" + expr + "，" + err.Error())
		}
	}

	fs, err := m_publicFiles.SearchPath(path, rx)
	if err != nil {
		return 605, errors.New("文件夹路径错误：" + err.Error())
	}
	return fs, nil
}

//Get：graph/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//404：资源不存在
//500：服务器错误
func GetGraph(ub *usrDb, r *http.Request, gf *GetGraphForm) (interface{}, error) {
	//ub := ctx.Value("UsrDb").(usrDb)
	path := r.URL.Path[len("/api/graph"):]

	//组装查询语句
	var query string = "path like ? "
	var timeMin int64
	tNow := time.Now().Unix()

	if gf.Flag == "Day" {
		timeMin = tNow - int64(gf.Day*gf.Limit)*86400
		query += " and time > ? and time < ? "
	} else {
		if gf.Year != 0 {
			query += " and year = ? "
		} else {
			query += `and 0 = ? `
		}
	}

	//根据标志分组统计结果
	var rows *sql.Rows
	var err error
	switch gf.Flag {
	case "Year":
		rows, err = global.Ds.Query("select year, count(*) cnt from downloads where "+query+" group by year ORDER BY year",
			path+"%", gf.Year)
	case "Month":
		rows, err = global.Ds.Query("select month, count(*) cnt from downloads where month > 0 and month < 13 and "+query+" group by month ORDER BY month",
			path+"%", gf.Year)
	case "Weekday":
		rows, err = global.Ds.Query("select weekday, count(*) cnt from downloads where weekday >= 0 and weekday < 7 and "+query+" group by weekday ORDER BY weekday",
			path+"%", gf.Year)
	default: //"Day"
		rows, err = global.Ds.Query("select count(*) cnt, min(time) from downloads where "+query+" group by year, month, day ORDER BY time",
			path+"%", timeMin, tNow)
	}

	var ghs GraphInfos
	ghs.Flag = gf.Flag
	var tmp Point
	if err != nil {
		goto ERROR500
	}
	defer rows.Close() //必须关闭，否则其他操作被lock

	//获取最终结果
	switch gf.Flag {
	case "Year":
		for rows.Next() {
			err = rows.Scan(&tmp.Year, &tmp.Cnt)
			if err != nil {
				goto ERROR500
			}
			ghs.PointList = append(ghs.PointList, tmp)
		}
	case "Month":
		ghs.PointList = make([]Point, 12)
		for i := 0; i < 12; i++ {
			ghs.PointList[i].Month = i + 1
		}
		for rows.Next() {
			err = rows.Scan(&tmp.Month, &tmp.Cnt)
			if err != nil {
				goto ERROR500
			}
			ghs.PointList[tmp.Month-1].Cnt = tmp.Cnt
		}
	case "Weekday":
		ghs.PointList = make([]Point, 7)
		for i := 0; i < 7; i++ {
			ghs.PointList[i].Weekday = i
		}
		for rows.Next() {
			err = rows.Scan(&tmp.Weekday, &tmp.Cnt)
			if err != nil {
				goto ERROR500
			}
			ghs.PointList[tmp.Weekday].Cnt = tmp.Cnt
		}
	default: //"Day"
		ghs.PointList = make([]Point, gf.Limit)
		for i := 0; i < gf.Limit; i++ {
			tNext := time.Unix(tNow-int64((gf.Limit-1-i)*gf.Day)*86400, 0)
			y, m, d := tNext.Date()
			tmp.Year, tmp.Month, tmp.Day = y, int(m), d
			tmp.Cnt = 0
			ghs.PointList[i] = tmp
		}

		var cnt int
		var t int64
		for rows.Next() {
			err = rows.Scan(&cnt, &t)
			if err != nil {
				goto ERROR500
			}

			index := gf.Limit - int((tNow-t)/86400/int64(gf.Day))
			ghs.PointList[index-1].Cnt += cnt
		}
	}

	return ghs, nil

ERROR500:
	return 500, fmt.Errorf("uid为%v的用户查询下载统计信息失败，查询条件为%s，错误信息为%s", ub.Uid, query, err.Error())
}

//Get：cnt/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//404：资源不存在
//500：服务器错误
func GetCnt(ub *usrDb, r *http.Request, gf *GetCntForm) (interface{}, error) {
	//ub := ctx.Value("UsrDb").(usrDb)

	path := r.URL.Path[len("/api/cnt"):]

	//组装查询语句
	var query string = "path like ? "
	if gf.Class+gf.Department+gf.RealName != "" {
		query += "and uid in ( select rowid from users where "
		if gf.Class != "" {
			query += "class = ? "
		} else {
			query += `"" = ? `
		}
		if gf.Department != "" {
			query += "and department = ? "
		} else {
			query += `and "" = ? `
		}
		if gf.RealName != "" {
			query += "and real_name like ? )"
		} else {
			query += `and "%%" = ? )`
		}
	} else {
		query += `and "" = ? `
		query += `and "" = ? `
		query += `and "%%" = ? `
	}

	var timeMin int64
	if gf.Day > 0 {
		timeMin = time.Now().Unix() - int64(gf.Day)*86400
		query += " and time > ? "
	} else {
		query += `and 1 > ? `
	}

	//第1页查询时给出总共有多少条记录
	var cts CntInfos
	var tmp CntInfo
	var rows *sql.Rows
	cts.CntList = make([]CntInfo, 0)
	var err error

	cts.Sum, err = global.Ds.Numerical("select count(distinct path) from downloads where "+query, path+"%", gf.Class, gf.Department, "%"+gf.RealName+"%", timeMin)
	if err != nil {
		goto ERROR500
	}

	rows, err = global.Ds.Query("select path, count(*) cnt, max(time) from downloads group by path having "+query+" ORDER BY cnt desc limit ? offset ? ",
		path+"%", gf.Class, gf.Department, "%"+gf.RealName+"%", timeMin, gf.OnePageCount, gf.OnePageCount*(gf.Page-1))
	if err != nil {
		goto ERROR500
	}
	defer rows.Close() //必须关闭，否则其他操作被lock

	for rows.Next() {
		err = rows.Scan(&tmp.Path, &tmp.Cnt, &tmp.Time)
		if err != nil {
			goto ERROR500
		}
		if fi, err := m_publicFiles.Stat(tmp.Path); err != nil {
			tmp.FileSize = -1
		} else {
			tmp.FileSize = fi.FileSize
			tmp.IsDir = fi.IsDir
		}
		cts.CntList = append(cts.CntList, tmp)
	}
	return cts, nil

ERROR500:
	return 500, fmt.Errorf("uid为%v的用户查询下载统计信息失败，查询条件为%s，错误信息为%s", ub.Uid, query, err.Error())
}

//Get：downloads/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//404：资源不存在
//500：服务器错误
func GetDownloads(ub *usrDb, r *http.Request, gf *GetCntForm) (interface{}, error) {
	//ub := ctx.Value("UsrDb").(usrDb)

	path := r.URL.Path[len("/api/downloads"):]

	var query string = "where path like ? "
	if gf.Class != "" {
		query += "and class = ? "
	} else {
		query += `and "" = ? `
	}
	if gf.Department != "" {
		query += "and department = ? "
	} else {
		query += `and "" = ? `
	}
	if gf.RealName != "" {
		query += "and real_name like ? "
	} else {
		query += `and "%%" = ? `
	}

	//第1页查询时给出总共有多少条记录
	var dfs DownloadInfos
	dfs.DownloadList = make([]downloadDb, 0)
	var err error

	dfs.Sum, err = global.Ds.Numerical("select count(*) from downloads inner join users on downloads.uid = users.rowid "+
		query, path+"%", gf.Class, gf.Department, "%"+gf.RealName+"%")
	if err != nil {
		goto ERROR500
	}

	err = global.Ds.LoadRow(&dfs.DownloadList, query+" ORDER BY time desc limit ? offset ? ",
		path+"%", gf.Class, gf.Department, "%"+gf.RealName+"%", gf.OnePageCount, gf.OnePageCount*(gf.Page-1))
	if err != nil {
		goto ERROR500
	}

	for i := 0; i < len(dfs.DownloadList); i++ {
		if fi, err := m_publicFiles.Stat(dfs.DownloadList[i].Path); err != nil {
			dfs.DownloadList[i].FileSize = -1
		} else {
			dfs.DownloadList[i].FileSize = fi.FileSize
			dfs.DownloadList[i].IsDir = fi.IsDir
		}
	}

	return dfs, nil

ERROR500:
	return 500, fmt.Errorf("uid为%v的用户查询下载统计信息失败，查询条件为%s，错误信息为%s", ub.Uid, query, err.Error())
}

//Get：dir/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//404：资源不存在
//500：服务器错误
func ViewPath(r *http.Request) (interface{}, error) {
	path := r.URL.Path[len("/api/dir"):]
	fs, err := m_publicFiles.ViewPath(path)
	if err != nil {
		return 605, errors.New("文件夹路径错误：" + err.Error())
	}
	return fs, nil
}

//Get：archive/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//404：资源不存在
//500：服务器错误
func DownloadArchive(u *usrDb, w http.ResponseWriter, r *http.Request) (interface{}, error) {
	//u := ctx.Value("UsrDb").(usrDb)

	path := r.URL.Path[len("/api/archive"):]
	err := m_publicFiles.GetArchive(w, r, path, ".zip", global.Conf.MaxDownloadSize)
	if err != nil {
		return 605, errors.New("下载压缩文件错误：" + err.Error())
	}

	//下载记录存入数据库
	var tmp downDb
	tmp.Uid = u.Uid
	tmp.Path = path
	t := time.Now()
	tmp.Time = t.Unix()
	y, m, d := t.Date()
	tmp.Year, tmp.Month, tmp.Day = y, int(m), d
	tmp.Weekday = int(t.Weekday())
	tmp.Ip, _ = IPv4(r.RemoteAddr)

	//插入数据库
	global.Ds.Insert(&tmp)
	return nil, nil
}

//Get：file/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//404：资源不存在
//500：服务器错误
func DownloadFile(u *usrDb, w http.ResponseWriter, r *http.Request) (interface{}, error) {
	//u := ctx.Value("UsrDb").(usrDb)

	path := r.URL.Path[len("/api/file"):]
	err := m_publicFiles.Download(w, r, path)
	if err != nil {
		return 605, errors.New("文件路径错误：" + err.Error())
	}
	//下载记录存入数据库
	var tmp downDb
	tmp.Uid = u.Uid
	tmp.Path = path
	t := time.Now()
	tmp.Time = t.Unix()
	y, m, d := t.Date()
	tmp.Year, tmp.Month, tmp.Day = y, int(m), d
	tmp.Weekday = int(t.Weekday())
	tmp.Ip, _ = IPv4(r.RemoteAddr)

	//插入数据库
	global.Ds.Insert(&tmp)

	//测试数据
	//	for i := 0; i < 365; i++ {
	//		tt := t.Add(time.Duration(int64(time.Hour) * 24 * int64(-1-i)))
	//		tmp.Time = tt.Unix()
	//		y, m, d := tt.Date()
	//		tmp.Year, tmp.Month, tmp.Day = y, int(m), d
	//		tmp.Weekday = int(tt.Weekday())
	//		global.Ds.Insert(&tmp)
	//	}
	return nil, nil
}

//POST：archive/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
//604, "文件扩展名非法"
//605, "文件或文件夹路径错误"
func UploadArchive(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	if r.ContentLength > global.Conf.MaxUploadSize {
		return 400, fmt.Errorf("上传文件过大，长度为%d", r.ContentLength)
	}

	path, fileName := files.Split(r.URL.Path[len("/api/archive"):])
	//判断路径是否存在
	if !m_publicFiles.IsExist(path) {
		return 605, errors.New("上传文件的路径不存在：" + path)
	}
	//先判断一遍文件后缀名
	fileExt := strings.ToLower(filepath.Ext(fileName))
	if len(fileExt) == 0 || strings.Contains(fileExt, ",") || !strings.Contains(global.Conf.ArchiveTable, fileExt) {
		return 604, errors.New("文件扩展名非法：" + fileName)
	}

	//初步判断都做完了再读取body
	r.Body = http.MaxBytesReader(w, r.Body, global.Conf.MaxUploadSize)
	file, head, err := r.FormFile("File")
	if err != nil {
		return 400, errors.New("FormFile: " + err.Error())
	}
	defer file.Close()

	//检查实际文件扩展名是否合法
	fileName = head.Filename
	fileExt = strings.ToLower(filepath.Ext(fileName))
	if len(fileExt) == 0 || strings.Contains(fileExt, ",") || !strings.Contains(global.Conf.ArchiveTable, fileExt) {
		return 604, errors.New("压缩文件扩展名非法：" + fileName)
	}

	//创建文件
	err = m_publicFiles.AddArchive(fileName, file, path)
	if err != nil {
		return 605, errors.New("上传压缩文件错误：" + err.Error())
	}
	return fileName, nil
}

//POST：file/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
//604, "文件扩展名非法"
//605, "文件或文件夹路径错误"
func UploadFile(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	if r.ContentLength > global.Conf.MaxUploadSize {
		return 400, fmt.Errorf("上传文件过大，长度为%d", r.ContentLength)
	}
	path, fileName := files.Split(r.URL.Path[len("/api/file"):])

	//判断路径是否存在
	if !m_publicFiles.IsExist(path) {
		return 605, errors.New("上传文件的路径不存在：" + path)
	}
	//先判断一遍文件后缀名
	fileExt := strings.ToLower(filepath.Ext(fileName))
	if strings.Contains(fileExt, ",") || !strings.Contains(global.Conf.ExtTable, fileExt) {
		return 604, errors.New("文件扩展名非法：" + fileName)
	}

	r.Body = http.MaxBytesReader(w, r.Body, global.Conf.MaxUploadSize)
	file, head, err := r.FormFile("File")
	if err != nil {
		return 400, errors.New("FormFile: " + err.Error())
	}
	defer file.Close()

	//检查文件扩展名是否合法
	fileName = head.Filename
	fileExt = strings.ToLower(filepath.Ext(fileName))
	//扩展名可以为空
	if strings.Contains(fileExt, ",") || !strings.Contains(global.Conf.ExtTable, fileExt) {
		return 604, errors.New("文件扩展名非法：" + fileName)
	}

	//创建文件
	err = m_publicFiles.AddFile(filepath.Join(path, fileName), file)
	if err != nil {
		return 605, errors.New("文件路径错误：" + err.Error())
	}
	return fileName, nil
}

//DELETE：file/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
//605："文件或文件夹路径错误"
func RemoveFile(u *usrDb, r *http.Request) (interface{}, error) {
	path := r.URL.Path[len("/api/file"):]

	//u := ctx.Value("UsrDb").(usrDb)

	if u.Class == "系统管理员" {
		//删除文件夹下所有内容
		err := m_publicFiles.RemoveAll(path)
		if err != nil {
			return 605, errors.New("文件夹路径错误：" + err.Error())
		}
		return 200, nil
	}

	//删除文件或者空文件夹
	err := m_publicFiles.Remove(path)
	if err != nil {
		return 605, errors.New("文件路径错误：" + err.Error())
	}
	return 200, nil
}

//DELETE：dir/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
//605："文件或文件夹路径错误"
func RemoveDir(r *http.Request) (interface{}, error) {
	path := r.URL.Path[len("/api/dir"):]

	//删除文件夹
	err := m_publicFiles.RemoveAll(path)
	if err != nil {
		return 605, errors.New("文件夹路径错误：" + err.Error())
	}
	return 200, nil
}

//POST：dir/{path}
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
//605："文件或文件夹路径错误"
func AddDir(r *http.Request) (interface{}, error) {
	path := r.URL.Path[len("/api/dir"):]

	//添加文件夹
	err := m_publicFiles.AddDir(path)
	if err != nil {
		return 605, errors.New("文件夹路径错误：" + err.Error())
	}
	return 200, nil
}

//PUT：dir/{path}  表单：NewName
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
//605："文件或文件夹路径错误"
func RenameDir(r *http.Request) (interface{}, error) {
	path := r.URL.Path[len("/api/dir"):]
	newName := r.FormValue("NewName")
	//检查名称合法性
	if newName == "" {
		return 400, errors.New("新文件夹名不能为空")
	}

	newPath := path[:strings.LastIndex(path, "/")] + "/" + newName

	//重命名文件
	err := m_publicFiles.Rename(path, newPath)
	if err != nil {
		return 605, errors.New("文件夹路径错误：" + err.Error())
	}
	return 200, nil
}

//PUT：file/{path}  表单：NewName
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
//604, "文件扩展名非法"
//605："文件或文件夹路径错误"
func RenameFile(r *http.Request) (interface{}, error) {
	path := r.URL.Path[len("/api/file"):]
	newName := r.FormValue("NewName")
	//检查名称合法性
	if newName == "" {
		return 400, errors.New("新文件夹名不能为空")
	}

	//检查文件扩展名是否合法
	var index = strings.LastIndex(newName, ".")
	var fileExt string = "dat" //扩展名可以为空
	if index >= 0 {
		fileExt = strings.ToLower(newName[strings.LastIndex(newName, ".")+1:])
	}

	if strings.Contains(fileExt, ",") || !strings.Contains(global.Conf.ExtTable, fileExt) {
		return 604, errors.New("文件扩展名非法：" + newName)
	}

	newPath := path[:strings.LastIndex(path, "/")] + "/" + newName
	//重命名文件
	err := m_publicFiles.Rename(path, newPath)
	if err != nil {
		return 605, errors.New("文件路径错误：" + err.Error())
	}
	return 200, nil
}
