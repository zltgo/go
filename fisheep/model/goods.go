package model

import (
	"errors"
	"net/http"
	"path"
	"time"

	"github.com/zltgo/api"
	"github.com/zltgo/node"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 商品属性，如产地，保存方式、生产日期等
type Attribute struct {
	Name  string `validate:"name,max=150"`
	Value string `validate:"omitempty,max=150"`
}

// collection: products
type GoodsDb struct {
	Id bson.ObjectId `bson:"_id"`
	//标题
	Title string `validate:"required,max=150"`
	//描述，宣传用如"产地直采/脆嫩香甜"
	Describe string `validate:"required,max=150"`

	//单位：分
	Price int `validate:"min=100,max=100000000"`
	//规格，“12个/4盒/2斤”等
	Weight string `validate:"omitempty,max=150"`
	Sold   int    // 已售
	Store  int    // 库存

	// 商品属性
	Attributes []Attribute `validate:"omitempty,max=16,dive"`
	IssueAt    time.Time
	// 销售区域ID
	AreaId string `validate:"hexadecimal,len=24"`
	// 产品分类ID
	ProductId string `validate:"hexadecimal,len=24"`
	// 1:下架，2 上架
	Flag int `validate:"min=1,max=2"`
	//排序用
	Order string `validate:"omitempty,alphanum,max=32"`
}

// 添加一个商品：POST:/goods
// 错误码：400,401,403,429,500,901,902
// 输入：goodsDb
// 返回值：新增的商品ID
func AddGoods(db *mgo.Database, mb *ManagerDb, gb *GoodsDb) (interface{}, error) {
	// 检查管理员的区域操作权限
	code, err := checkArea(db, mb, gb.AreaId)
	if code != http.StatusOK {
		return code, err
	}
	gb.Id = bson.NewObjectId()
	gb.IssueAt = time.Now()

	if err := db.C(Goods).Insert(gb); err != nil {
		if mgo.IsDup(err) {
			return StatusDbDup, err
		}
		return http.StatusInternalServerError, err
	}

	return gb.Id, nil
}

// 修改商品：PUT:/goods
// 错误码：400,401,403,429,500,900,901,902
// 输入：goodsDb
// 返回值：<nil>
func ModifyGoods(db *mgo.Database, mb *ManagerDb, gb *GoodsDb) (int, error) {
	// 有可能为空
	if gb.Id.Valid() == false {
		return http.StatusBadRequest, errors.New("goodsDb.Id is not valid")
	}
	// 检查管理员的区域操作权限
	code, err := checkArea(db, mb, gb.AreaId)
	if code != http.StatusOK {
		return code, err
	}

	if err := db.C(Goods).UpdateId(gb.Id, gb); err != nil {
		if err == mgo.ErrNotFound {
			return StatusDbNotFound, err
		}
		if mgo.IsDup(err) {
			return StatusDbDup, err
		}
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

// 查找商品时提交的表单
type GetGoodsForm struct {
	// 销售区域ID
	AreaId string `bson:",omitempty" validate:"omitempty,hexadecimal,len=24"`
	// 商品分类ID
	ProductId string `bson:",omitempty" validate:"omitempty,hexadecimal,len=24"`

	Title *bson.RegEx `bson:",omitempty"`
	//标题，模糊匹配
	GoodsName string `bson:"-" validate:"omitempty,max=150"`

	// 1:下架，2 上架
	Flag int `bson:",omitempty" validate:"omitempty,min=1,max=2"`

	Sort []string `bson:"-" validate:"omitempty,max=3,dive,eq=price|eq=sold|eq=store|eq=issueat|eq=order|eq=-price|eq=-sold|eq=-store|eq=-issueat|eq=-order"`

	// 默认每页显示10个
	Cnt int `bson:"-" validate:"omitempty,min=1" default:"10"`
	// 不能超过100页，避免skip大量结果导致性能问题
	Page int `bson:"-" validate:"omitempty,max=100"`
}

// 查找商品：GET:/goods
// 错误码：400,401,403,429,500
// 输入：GetGoodsForm
// 返回值：[]goodsDb
func GetGoods(db *mgo.Database, gf *GetGoodsForm) (interface{}, error) {
	// 构建正则表达式，类似于 like %Usr%
	if gf.GoodsName != "" {
		gf.Title = &bson.RegEx{gf.GoodsName, "i"}
	}

	var gs []GoodsDb
	if err := db.C(Goods).Find(gf).Limit(gf.Cnt).Skip(gf.Page * gf.Cnt).Sort(gf.Sort...).All(&gs); err != nil {
		if err == mgo.ErrNotFound {
			return http.StatusOK, nil
		}
		return http.StatusInternalServerError, err
	}

	return gs, nil
}

// 删除商品：DELETE:/goods/:id
// 错误码：400,401,403,429,500,900,902
// 输入：无
// 返回值：<nil>
func RemoveGoods(db *mgo.Database, mb *ManagerDb, ctx *api.Context) (int, error) {
	id := ctx.Params.Get("id")
	if !bson.IsObjectIdHex(id) {
		return http.StatusBadRequest, errors.New("id of goods is not valid")
	}

	var gb GoodsDb
	if err := db.C(Goods).FindId(bson.ObjectIdHex(id)).One(&gb); err != nil {
		if err == mgo.ErrNotFound {
			return StatusDbNotFound, nil
		}
		return http.StatusInternalServerError, err
	}

	// 检查管理员的区域操作权限
	code, err := checkArea(db, mb, gb.AreaId)
	if code != http.StatusOK {
		return code, err
	}

	if err := db.C(Goods).RemoveId(bson.ObjectIdHex(id)); err != nil {
		if err == mgo.ErrNotFound {
			return StatusDbNotFound, err
		}
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

// 判断区域aid是否在manager的管辖范围内
func checkArea(db *mgo.Database, mb *ManagerDb, aid string) (int, error) {
	// 相等即可，大多数情况不用读数据库
	if aid == mb.AreaId {
		return http.StatusOK, nil
	}
	nd, err := node.GetById(db.C(Areas), aid)
	if err != nil {
		if err == mgo.ErrNotFound {
			return StatusAreaErr, err
		} else {
			return http.StatusInternalServerError, err
		}
	}
	// 判断mb.AreaId是否在nd.Ancestors中
	for _, id := range nd.Ancestors {
		if id == mb.AreaId {
			return http.StatusOK, nil
		}
	}
	return StatusAreaErr, errors.New(mb.Usr + "没有权限操作区域：" + path.Join(nd.Dir, nd.Name))
}
