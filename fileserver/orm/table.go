package orm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	. "github.com/zltgo/fileserver/utils" //前面加'.'表示调用包函数时可以省略包名
)

//声明一个函数类型
type table struct {
	strucType   reflect.Type                      //结构体类型
	keyIndexs   []int                             //存储结构体中与数据库对应的属性序号,包括pk
	pkIndexs    []int                             //存储结构体中与数据库主键对应的属性序号
	selectStr   string                            //查询语句
	selectPkStr string                            //查询语句,通过主键
	insertStr   string                            //插入语句
	replaceStr  string                            //保存语句
	updatePkStr string                            //更新语句,通过主键
	updateStr   string                            //更新语句，通过sql语句
	deleteStr   string                            //删除语句
	pkCond      string                            //主键条件
	auto2If     func(reflect.Value) []interface{} //函数对象
}

type iDb interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

/*
设置要操作的表格以及对应的结构体类型，
该操作是非线程安全的，最好在程序初始化时设置完毕
结构体的定义如下(不支持嵌套)
type RAE struc {
	Id    int64 `PK:"uid"` //PK表示主键，字段名为"uid"
	R     int   `距离`//对应表格中的字段“距离”
	A     int   `方位` //同上
	E     int    `俯仰` //同上
}
*/
func NewTable(tbname string, strucType reflect.Type) *table {
	Assert(tbname != "", "表名不能为空")
	Assert(strucType.Kind() == reflect.Struct, "只能使用struct类型来与表格[%s]进行绑定", tbname)

	tb := &table{strucType: strucType}
	//解析结构体,   有tag的属性表示要跟数据库字段对应
	var keys []string       //非主键集合
	var pks []string        //主键集合
	var keyHolders []string //非主键集合, 带 “=？”
	var pkHolders []string  //主键集合，带 “=？”
	var holders []string
	for i := 0; i < strucType.NumField(); i++ {
		tag := strucType.Field(i).Tag
		if len(tag) != 0 {
			key := tag.Get("PK")
			if key != "" {
				pks = append(pks, key)
				pkHolders = append(pkHolders, key+" = ?")
				tb.pkIndexs = append(tb.pkIndexs, i)
			} else {
				key = reflect.ValueOf(tag).String()
			}
			keys = append(keys, key)
			keyHolders = append(keyHolders, key+" = ?")
			tb.keyIndexs = append(tb.keyIndexs, i)
			holders = append(holders, "?")
		}
	}

	Assert(len(keys) != 0, "%v中未找到与数据库%s对应的字段", strucType, tbname)

	var fileds string = strings.Join(keys, ", ")
	var holderstr string = strings.Join(holders, ", ")
	var condstr string = strings.Join(pkHolders, " and ")
	tb.selectStr = fmt.Sprintf("select %s from %s ", fileds, tbname)
	tb.insertStr = fmt.Sprintf("insert into %s(%s)  values(%s) ", tbname, fileds, holderstr)
	tb.replaceStr = fmt.Sprintf("replace into %s(%s)  values(%s) ", tbname, fileds, holderstr)
	tb.updatePkStr = fmt.Sprintf("update  %s set %s where %s ", tbname, strings.Join(keyHolders, ", "), condstr)
	tb.updateStr = fmt.Sprintf("update  %s set %s ", tbname, strings.Join(keyHolders, ", "))
	tb.selectPkStr = tb.selectStr + "where " + condstr

	//没有主键则使用所有值作为删除条件
	tb.auto2If = tb.pk2If
	if len(pkHolders) == 0 {
		condstr = strings.Join(keyHolders, " and ")
		tb.auto2If = tb.key2If
	}
	tb.deleteStr = fmt.Sprintf("delete from  %s where %s", tbname, condstr)

	return tb
}

//将结构体的数据库字段转为接口数组
func (m table) key2If(v reflect.Value) []interface{} {
	//获取结构体中与数据库有关联的字段
	rv := make([]interface{}, len(m.keyIndexs))
	for i, index := range m.keyIndexs {
		rv[i] = v.Field(index).Interface()
	}
	return rv
}

//将结构体的主键字段转为接口数组
func (m table) pk2If(v reflect.Value) []interface{} {
	rv := make([]interface{}, len(m.keyIndexs))
	for i, index := range m.keyIndexs {
		rv[i] = v.Field(index).Interface()
	}
	return rv
}

//将结构体的数据库字段转为接口地址数组，load函数回写记录时需要用到
//是原输入参数是结构体的值的话，不能调用该函数的，会painic: can't addr
func (m table) key2IfAddr(v reflect.Value) []interface{} {
	//获取结构体中与数据库有关联的字段
	rv := make([]interface{}, len(m.keyIndexs))
	for i, index := range m.keyIndexs {
		rv[i] = v.Field(index).Addr().Interface()
	}
	return rv
}

//插入记录，输入参数为struct的值、指针、Slice（值或指针）
func (m table) insert(db iDb, v reflect.Value) error {
	//获取参数类型
	k := v.Type().Kind()

	if k == reflect.Struct {
		_, err := db.Exec(m.insertStr, m.key2If(v)...)
		return err
	}

	//除了struct类型就是slice类型
	for i := 0; i < v.Len(); i++ {
		_, err := db.Exec(m.insertStr, m.key2If(v.Index(i))...)
		if err != nil {
			return err
		}
	}
	return nil
}

//检查rowsAffected，必须大于0，否则返回错误"sql: no rows in result set"
func checkRowsAffected(res sql.Result, err error) error {
	if err == nil {
		var cnt int64
		cnt, err = res.RowsAffected()
		if err == nil && cnt == 0 {
			err = sql.ErrNoRows
		}
	}
	return err
}

//更新记录，输入参数为struct的值、指针、Slice（值或指针）
//结构体必须有对应的主键字段，主键的值是更新的唯一条件。
//没有记录满足条件时，会返回错误"sql: no rows in result set"。
func (m table) update(db iDb, v reflect.Value) error {
	//判断是否有主键
	if len(m.pkIndexs) == 0 {
		return fmt.Errorf("orm: 结构体%v中未找到主键字段", m.strucType)
	}

	//获取参数类型
	k := v.Type().Kind()

	if k == reflect.Struct {
		tmp := append(m.key2If(v), m.pk2If(v)...)
		res, err := db.Exec(m.updatePkStr, tmp...)
		return checkRowsAffected(res, err)
	}

	//除了struct类型就是slice类型
	for i := 0; i < v.Len(); i++ {
		tmp := append(m.key2If(v.Index(i)), m.pk2If(v.Index(i))...)
		res, err := db.Exec(m.updatePkStr, tmp...)
		err = checkRowsAffected(res, err)
		if err != nil {
			return err
		}
	}
	return nil
}

//更新记录，输入参数为struct的值、指针，以输入的sql语句作为更新条件，
//没有记录满足条件时，会返回错误"sql: no rows in result set"。
func (m table) updateRow(db iDb, v reflect.Value, query string, args ...interface{}) error {
	//获取参数类型
	k := v.Type().Kind()
	if k != reflect.Struct {
		return fmt.Errorf("orm: 输入参数的类型只能为Struct，实际为%v", k)
	}
	query = m.updateStr + query
	tmp := append(m.key2If(v), args...)
	res, err := db.Exec(query, tmp...)
	return checkRowsAffected(res, err)
}

//删除记录，输入参数为struct的值、指针、Slice（值或指针）
//有主键使用主键作为条件，否则使用所有字段作为条件
func (m table) remove(db iDb, v reflect.Value) error {
	//获取参数类型
	k := v.Type().Kind()

	if k == reflect.Struct {
		res, err := db.Exec(m.deleteStr, m.auto2If(v)...)
		return checkRowsAffected(res, err)
	}

	//除了struct类型就是slice类型
	for i := 0; i < v.Len(); i++ {
		res, err := db.Exec(m.deleteStr, m.auto2If(v.Index(i))...)
		err = checkRowsAffected(res, err)
		if err != nil {
			return err
		}
	}
	return nil
}

//保存记录，相当于insert or update，输入参数为struct的值、指针、Slice（值或指针）
func (m table) save(db iDb, v reflect.Value) error {
	//获取参数类型
	k := v.Type().Kind()

	if k == reflect.Struct {
		_, err := db.Exec(m.replaceStr, m.key2If(v)...)
		return err
	}

	//除了struct类型就是slice类型
	for i := 0; i < v.Len(); i++ {
		_, err := db.Exec(m.replaceStr, m.key2If(v.Index(i))...)
		if err != nil {
			return err
		}
	}
	return nil
}

//读取记录，输入参数为struct的指针、Slice的指针
//以主键作为查询条件
func (m table) load(db iDb, v reflect.Value) error {
	//判断是否有主键
	if len(m.pkIndexs) == 0 {
		return fmt.Errorf("orm: 结构体%v中未找到主键字段", m.strucType)
	}

	//获取参数类型
	k := v.Type().Kind()

	if k == reflect.Struct {
		row := db.QueryRow(m.selectPkStr, m.pk2If(v)...) //该接口最多返回一条
		return row.Scan(m.key2IfAddr(v)...)
	}

	//除了struct类型就是slice类型
	for i := 0; i < v.Len(); i++ {
		row := db.QueryRow(m.selectPkStr, m.pk2If(v.Index(i))...) //该接口最多返回一条
		err := row.Scan(m.key2IfAddr(v.Index(i))...)
		if err != nil {
			return err
		}
	}
	return nil
}

//读取记录，输入参数为struct的指针、Slice的指针
//以输入的sql语句作为查询条件，query中的？只能是字段的值，不能是字段的名称，
//例如（where ? > ?, age, 10)是不对的
func (m table) loadRow(db iDb, v reflect.Value, query string, args ...interface{}) error {
	//获取参数类型
	k := v.Type().Kind()

	query = m.selectStr + query
	if k == reflect.Struct {
		row := db.QueryRow(query, args...) //该接口最多返回一条,没有会报错ErrNoRows
		return row.Scan(m.key2IfAddr(v)...)
	}

	//除了struct类型就是slice类型，先清空
	newSlice := reflect.MakeSlice(v.Type(), 0, 100)
	v.Set(newSlice)
	rows, err := db.Query(query, args...) //该接口返回多条，没有不会报错
	if err != nil {
		return err
	}
	defer rows.Close() //必须关闭，否则其他操作被lock

	tmp := reflect.New(m.strucType)
	for rows.Next() {
		err = rows.Scan(m.key2IfAddr(tmp.Elem())...)
		if err != nil {
			return err
		}
		newSlice = reflect.Append(newSlice, tmp.Elem())
	}
	//输出参数赋值
	v.Set(newSlice)
	return nil
}
