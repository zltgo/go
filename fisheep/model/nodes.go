package model

import (
	"errors"
	"net/http"

	"github.com/zltgo/node"
	"gopkg.in/mgo.v2"
)

type Nodes string

// 增加一个节点：POST:/area, POST:/product
// 错误码：400,401,403,500,900,901
// 输入：NodeForm
// 返回值：Node
// type Node struct {
//	 Id   string `bson:"_id"`
//	 Pid  string // parent id
//	 Name string
//	 Dir  string
// }
// 说明：Pid为空时表示添加到根目录下
func (m Nodes) Add(db *mgo.Database, nf *NodeForm) (interface{}, error) {
	if nf.Pid == "" {
		nf.Pid = node.RootId
	}
	nd, err := node.Insert(db.C(string(m)), nf.Pid, nf.Name)

	switch {
	case err == node.ErrName:
		return http.StatusBadRequest, err
	case err == mgo.ErrNotFound:
		return StatusDbNotFound, err
	case mgo.IsDup(err):
		return StatusDbDup, err
	case err != nil:
		return http.StatusInternalServerError, err
	}
	return nd, nil
}

// 修改节点名称：PUT:/area/name, PUT:/product/name
// 错误码：400,401,403,500,900,901
// 输入：NodeForm
// 返回值：<nil>
func (m Nodes) ResetName(db *mgo.Database, nf *NodeForm) (int, error) {
	if nf.Id == "" {
		return http.StatusBadRequest, errors.New("NodeForm.Id 不能为空")
	}
	err := node.Rename(db.C(string(m)), nf.Id, nf.Name)

	switch {
	case err == node.ErrName:
		return http.StatusBadRequest, err
	case err == mgo.ErrNotFound:
		return StatusDbNotFound, err
	case mgo.IsDup(err):
		return StatusDbDup, err
	case err != nil:
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

// 重新设置父节点：PUT:/area/pid, PUT:/product/pid
// 错误码：400,401,403,500,900,901
// 输入：NodeForm
// 返回值：<nil>
// 说明：Pid为空时表示设置到根目录下
func (m Nodes) ResetPid(db *mgo.Database, nf *NodeForm) (int, error) {
	if nf.Id == "" {
		return http.StatusBadRequest, errors.New("NodeForm.Id 不能为空")
	}
	if nf.Pid == "" {
		nf.Pid = node.RootId
	}
	err := node.ResetPid(db.C(string(m)), nf.Id, nf.Pid)

	switch {
	case err == mgo.ErrNotFound:
		return StatusDbNotFound, err
	case mgo.IsDup(err):
		return StatusDbDup, err
	case err != nil:
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

// 获取子节点：GET:/areas, GET:/products
// 错误码：400,401,403,500,900
// 输入：NodeForm
// 返回值：[]Node
// type Node struct {
//	 Id   string `bson:"_id"`
//	 Pid  string // parent id
//	 Name string
//	 Dir  string
// }
// 说明：Id为空时表示获取根目录下的节点
func (m Nodes) GetChildren(db *mgo.Database, nf *NodeForm) (interface{}, error) {
	if nf.Id == "" {
		nf.Id = node.RootId
	}
	nds, err := node.GetChildren(db.C(string(m)), nf.Id)
	switch {
	case err == mgo.ErrNotFound:
		return StatusDbNotFound, err
	case err != nil:
		return http.StatusInternalServerError, err
	}
	return nds, nil
}

// 删除节点及其子孙：DELETE:/areas, DELETE:/products
// 错误码：400,401,403,500,900
// 输入：NodeForm
// 返回值：<nil>
func (m Nodes) Remove(db *mgo.Database, nf *NodeForm) (int, error) {
	err := node.RemoveId(db.C(string(m)), nf.Id)

	switch {
	case err == mgo.ErrNotFound:
		return StatusDbNotFound, err
	case err != nil:
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

// 修复所有节点关系：PUT:/areas, PUT:/products
// 错误码：400,401,403,500
// 输入：无
// 返回值：<nil>
// 用户数据库异常情况下修复节点关系
func (m Nodes) RebuildAll(db *mgo.Database) (int, error) {
	if err := node.RebuildAll(db.C(string(m))); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
