package orm

import (
	"database/sql"

	. "github.com/zltgo/fileserver/utils" //前面加'.'表示调用包函数时可以省略包名
)

type Session struct {
	*sql.Tx          //事务指针,匿名组合,可以直接用tx的所有接口
	ds      *DataSet //模型指针
}

//插入数据，输入参数可以为多种结构体的值、指针、数组，
//该操作是原子的，要么都成功，要么失败。
func (m *Session) Insert(args ...interface{}) error {
	for _, arg := range args {
		v := Any2Value(arg)
		tb, err := m.ds.findTable(v.Type())
		if err == nil {
			err = tb.insert(m.Tx, v)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

//更新记录，输入参数为struct的值、指针、Slice（值或指针），
//结构体必须有对应的主键字段，主键的值是更新的唯一条件。
//没有记录满足条件时，会返回错误"sql: no rows in result set"
func (m *Session) Update(args ...interface{}) error {
	for _, arg := range args {
		v := Any2Value(arg)
		tb, err := m.ds.findTable(v.Type())
		if err == nil {
			err = tb.update(m.Tx, v)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

//更新记录，输入参数为struct的值、指针，以输入的sql语句作为更新条件，
//没有记录满足条件时，会返回错误"sql: no rows in result set"
func (m *Session) UpdateRow(arg interface{}, query string, args ...interface{}) error {
	v := Any2Value(arg)
	tb, err := m.ds.findTable(v.Type())
	if err == nil {
		err = tb.updateRow(m.Tx, v, query, args...)
	}
	return err
}

//删除记录，输入参数为struct的值、指针、Slice（值或指针），
//有主键使用主键作为条件，否则使用所有字段作为条件。
//没有记录满足条件时，会返回错误"sql: no rows in result set"，所以
//输入参数中的主键不能重复且必须存在记录。
//DeleteRow自己用Exec接口即可。
func (m *Session) Delete(args ...interface{}) error {
	for _, arg := range args {
		v := Any2Value(arg)
		tb, err := m.ds.findTable(v.Type())
		if err == nil {
			err = tb.remove(m.Tx, v)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

//保存记录，相当于insert or update，输入参数为struct的值、指针、Slice（值或指针）。
func (m *Session) Save(args ...interface{}) error {
	for _, arg := range args {
		v := Any2Value(arg)
		tb, err := m.ds.findTable(v.Type())
		if err == nil {
			err = tb.save(m.Tx, v)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

//读取记录，输入参数为struct的指针、Slice的指针，
//以主键作为查询条件，查询结果回写到输入参数中。
func (m *Session) Load(args ...interface{}) error {
	for _, arg := range args {
		v := Ptr2Value(arg)
		tb, err := m.ds.findTable(v.Type())
		if err == nil {
			err = tb.load(m.Tx, v)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

//读取记录，输入参数为struct的指针、Slice的指针，
//以输入的sql语句作为查询条件，查询结果回写到输入参数中。
func (m *Session) LoadRow(arg interface{}, query string, args ...interface{}) error {
	v := Ptr2Value(arg)
	tb, err := m.ds.findTable(v.Type())
	if err == nil {
		err = tb.loadRow(m.Tx, v, query, args...)
	}
	return err
}
