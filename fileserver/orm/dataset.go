package orm

import (
	"database/sql"
	"fmt"
	"reflect"
	"runtime"

	. "github.com/zltgo/fileserver/utils" //前面加'.'表示调用包函数时可以省略包名
)

type DataSet struct {
	*sql.DB                                //数据库操作对象指针，全局一个即可，内部实现了连接池
	driverName     string                  //数据库类型，如mysql，sqlite3等
	dataSourceName string                  //数据源
	tableMap       map[reflect.Type]*table //保存表格与结构体的映射关系

}

func close(m *DataSet) {
	m.Close()
}

//新建一个数据库模型，参数与sql.Open相同
//程序中使用一个DataSet即可，可以在多个协程中使用
func NewDataSet(driverName, dataSourceName string) (*DataSet, error) {
	//检查是否可用
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	//设置db池可闲置个数，默认为100
	db.SetMaxIdleConns(100)

	//map, interfice,slice的默认值都为nil,但是map必须先make，
	//因为使用map[] = x时，map不能为nil
	ds := &DataSet{db, driverName, dataSourceName, make(map[reflect.Type]*table)}

	//设置析构函数
	runtime.SetFinalizer(ds, close)
	return ds, nil
}

/*
设置要操作的表格以及对应的结构体类型，
该操作是非线程安全的，最好在程序初始化时设置完毕。
结构体的定义如下(不支持嵌套)：
type RAE struc {
	Id    int64 `PK:"uid"` //PK表示主键，字段名为"uid"
	R     int   `距离`//对应表格中的字段“距离”
	A     int   `方位` //同上
	E     int    `俯仰` //同上
}
结构体名可以小写，字段名必须大写开头。
*/
func (m *DataSet) Register(tbname string, arg interface{}) {
	//获取结构体类型
	strucType := Any2Value(arg).Type()

	//创建表格对象并更新map
	tb := NewTable(tbname, strucType)
	m.tableMap[strucType] = tb
	m.tableMap[reflect.SliceOf(strucType)] = tb
	return
}

//查找table映射，参数类型只能为sturct及其slice，否则会找不到
func (m *DataSet) findTable(ty reflect.Type) (*table, error) {
	tb, ok := m.tableMap[ty]
	if !ok {
		return nil, fmt.Errorf("orm: 未找到%v对应的table，未使用dataset.SetTable进行绑定或数据类型不正确", ty)
	}
	return tb, nil
}

// 创建一个会话，执行数据库CURD，最后需Commit或者Rollback。
func (m *DataSet) NewSession() (*Session, error) {
	//创建sql.Tx指针
	tx, err := m.Begin()
	if err != nil {
		return nil, err
	}

	return &Session{tx, m}, nil
}

//插入数据，输入参数可以为多种结构体的值、指针、数组，
//该操作是原子的，要么都成功，要么失败。
func (m *DataSet) Insert(args ...interface{}) error {
	se, err := m.NewSession()
	if err != nil {
		return err
	}

	err = se.Insert(args...)
	if err == nil {
		err = se.Commit()
	} else {
		se.Rollback()
	}
	return err
}

//更新记录，输入参数为struct的值、指针、Slice（值或指针），
//结构体必须有对应的主键字段，主键的值是更新的唯一条件。
//该操作是原子的，不会出现更新一半的情况。
//没有记录满足条件时，会返回错误sql.ErrNoRows。
func (m *DataSet) Update(args ...interface{}) error {
	se, err := m.NewSession()
	if err != nil {
		return err
	}

	err = se.Update(args...)
	if err == nil {
		err = se.Commit()
	} else {
		se.Rollback()
	}
	return err
}

//更新记录，输入参数为struct的值、指针，以输入的sql语句作为更新条件,
//该操作是原子的，不会出现更新一半的情况。
//没有记录满足条件时，会返回错误sql.ErrNoRows。
func (m *DataSet) UpdateRow(arg interface{}, query string, args ...interface{}) error {
	v := Any2Value(arg)
	tb, err := m.findTable(v.Type())
	if err == nil {
		err = tb.updateRow(m.DB, v, query, args...)
	}
	return err
}

//删除记录，输入参数为struct的值、指针、Slice（值或指针），
//有主键使用主键作为条件，否则使用所有字段作为条件。
//没有记录满足条件时，会返回错误"sql: no rows in result set"，所以
//输入参数中的主键不能重复且必须存在记录。
func (m *DataSet) Delete(args ...interface{}) error {
	se, err := m.NewSession()
	if err != nil {
		return err
	}

	err = se.Delete(args...)
	if err == nil {
		err = se.Commit()
	} else {
		se.Rollback()
	}
	return err
}

//统计信息，例如select count(*) from table;max,ave,min等可以返回数字结果的sql语句
func (m *DataSet) Numerical(query string, args ...interface{}) (cnt int64, err error) {
	row := m.QueryRow(query, args...)
	err = row.Scan(&cnt)
	return
}

//保存记录，相当于insert or update，输入参数为struct的值、指针、Slice（值或指针）。
func (m *DataSet) Save(args ...interface{}) error {
	se, err := m.NewSession()
	if err != nil {
		return err
	}

	err = se.Save(args...)
	if err == nil {
		err = se.Commit()
	} else {
		se.Rollback()
	}
	return err
}

//读取记录，输入参数为struct的指针、Slice的指针，
//以主键作为查询条件。
func (m *DataSet) Load(args ...interface{}) error {
	se, err := m.NewSession()
	if err != nil {
		return err
	}

	err = se.Load(args...)
	if err == nil {
		err = se.Commit()
	} else {
		se.Rollback()
	}
	return err
}

//读取记录，输入参数为struct的指针、Slice的指针，
//以输入的sql语句作为查询条件，query中的？只能是字段的值，不能是字段的名称，
//例如（where ? > ?, age, 10)是不对的, 只能是（where age > ?, 10)
//因为替换？时，如果是字符串，则会加上引号'age'
func (m *DataSet) LoadRow(arg interface{}, query string, args ...interface{}) error {
	v := Ptr2Value(arg)
	tb, err := m.findTable(v.Type())
	if err == nil {
		err = tb.loadRow(m.DB, v, query, args...)
	}
	return err
}
