package model

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	. "fisheep/conf"

	"github.com/zltgo/api"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 上传商品图片：POST:/image/:id
// 错误码：400,401,403,429,500,900,902
// 输入：
// type UploadImageForm struct {
//	GoodsId string `validate:"hexadecimal,len=24"`
//	Name    string `validate:"name,min=2,max=64"`
// }
// 返回值：<nil>
// 注意事项：表单中的Name为上传后要保存的文件名，不包括后缀，后缀与上传时所选择的文件名保持一致
func UploadImage(db *mgo.Database, mb *ManagerDb, ctx *api.Context) (int, error) {
	id := ctx.Params.Get("id")
	if !bson.IsObjectIdHex(id) {
		return http.StatusBadRequest, errors.New("id of goods is not valid")
	}

	// 检查管理员是否有权限操作该商品
	var gb GoodsDb
	if err := db.C(Goods).FindId(bson.ObjectIdHex(id)).Select(bson.M{"areaid": 1}).One(&gb); err != nil {
		if err == mgo.ErrNotFound {
			return StatusDbNotFound, nil
		}
		return http.StatusInternalServerError, err
	}

	code, err := checkArea(db, mb, gb.AreaId)
	if code != http.StatusOK {
		return code, err
	}

	r := ctx.Request
	if r.ContentLength > Pri.MaxUploadSize {
		return http.StatusBadRequest, fmt.Errorf("上传文件不能大于%d，实际为%d", Pri.MaxUploadSize, r.ContentLength)
	}

	r.Body = http.MaxBytesReader(ctx.Writer(), r.Body, Pri.MaxUploadSize)
	file, head, err := r.FormFile("File")
	if err != nil {
		return http.StatusBadRequest, errors.New("FormFile: " + err.Error())
	}
	defer file.Close()

	//检查文件扩展名是否合法
	fileExt := strings.ToLower(filepath.Ext(head.Filename))
	if !strings.Contains(Pri.ExtTable, ","+fileExt+",") {
		return http.StatusBadRequest, errors.New("不支持的文件类型：" + head.Filename)
	}

	//创建文件夹，名字为商品ID
	dir := filepath.Join("./static/images", id)
	os.Mkdir(dir, os.ModePerm)

	//创建文件，已存在则替换
	f, err := os.OpenFile(filepath.Join(dir, head.Filename), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer f.Close()

	if _, err = io.Copy(f, file); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
