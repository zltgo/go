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

// 订单项
type OrderItem struct {
	GoodsId bson.ObjectId
	//商品数量
	Cnt int
	//单价，分
	Price int
}

// collection: products
type OrderDb struct {
	Id bson.ObjectId `bson:"_id"`
	// 商品列表
	Items []OrderItem `db:"-"`
}
