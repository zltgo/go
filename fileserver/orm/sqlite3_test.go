package orm

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	_ "github.com/go-sqlite3" //前面加'_'表示引入包（调用包的init），但是不引入包中的变量，函数等资源
)

/*
sqlite语法
1）创建数据库：
sqlite3 sqlite_test //创建名为sqlite_test的数据库文件
2）创建表格：
create table 'userinfo'(
	'uid' INTEGER PRIMARY KEY AUTOINCREMENT,
	'用户名' VARCHAR(64) DEFAULT NULL,
	'city' NCHAR(16) DEFAULT NULL,
	'age' INT(10) UNIQUE DEFAULT '0',
	'cash' DOUBLE DEFAULT '0',
	'sex' BOOLEAN DEFAULT 'FALSE',
	'date'DATE DEFAULT '2015-06-30',
	'time'TIME DEFAULT '00:00:00',
	'timestamp'TIMESTAMP DEFAULT '2015-06-30 00:00:00.000',
	'zip' BLOB NULL
);

注：调用sql.Open函数时，即使数据库不存在，也会创建一个数据库
*/
type UserInfo_noTag struct {
	m_tmp1      int //酱油1
	M_tmp2      int //酱油2
	M_username  string
	M_city      string
	M_age       int
	M_cash      float64
	M_sex       bool
	M_date      time.Time
	M_time      time.Time
	M_timestamp time.Time
	M_zip       []byte
}

type UserInfo struct {
	m_tmp1      int       //酱油1
	M_tmp2      int       //酱油2
	M_username  string    `用户名`
	M_city      string    `city`
	M_age       int       `age`
	M_cash      float64   `cash`
	M_sex       bool      `sex`
	M_timestamp time.Time `timestamp`
	M_zip       []byte    `zip`
}

type userInfoEx struct {
	M_id        int       `PK:"uid"`
	M_username  string    `用户名`
	M_city      string    `city`
	M_age       int       `age`
	M_cash      float64   `cash`
	M_sex       bool      `sex`
	M_timestamp time.Time `timestamp`
	M_zip       []byte    `zip`
}

type UserInfo_alias struct {
	m_tmp1      int       //酱油1
	M_tmp2      int       //酱油2
	M_username  string    `错误的字段名`
	M_city      string    `city`
	M_age       int       `age`
	M_cash      float64   `cash`
	M_sex       bool      `sex`
	M_date      time.Time `date`
	M_time      time.Time `time`
	M_timestamp time.Time `timestamp`
	M_zip       []byte    `zip`
}

type UserInfo_sport struct {
	m_tmp1      int       //酱油1
	M_tmp2      int       //酱油2
	M_username  string    `用户名`
	M_city      string    `city`
	M_age       int       `age`
	M_cash      float64   `cash`
	M_sex       bool      `sex`
	M_timestamp time.Time `timestamp`
	M_zip       []byte    `zip`
}

type cashAndSex struct {
	M_cash float64 `cash`
	M_sex  bool    `sex`
}

//测试空表名错误
func Test_SetTable_1(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if r != "表名不能为空" {
				t.Error(r)
			}

		}
	}()

	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
	}

	var struc UserInfo
	ds.Register("", struc)
}

//测试非sturct错误
func Test_SetTable_2(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if r != "只能使用struct类型来与表格[userinfo]进行绑定" {
				t.Error(r)
			}

		}
	}()

	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
	}

	var struc UserInfo
	ds.Register("userinfo", struc)
}

//测试sturct没有字段与数据库对应的错误
func Test_SetTable_3(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if r != "orm.UserInfo_noTag中未找到与数据库userinfo对应的字段" {
				t.Error(r)
			}

		}
	}()

	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
	}

	var struc UserInfo
	ds.Register("userinfo", struc)
}

//测试找不到struct
func Test_findSturct_1(t *testing.T) {
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
		return
	}

	var struc UserInfo
	//ds.Register("userinfo", struc)
	err = ds.Save(struc)
	errInfo := "orm: 未找到orm.UserInfo对应的table，未使用dataset.SetTable进行绑定或数据类型不正确"
	if err.Error() != errInfo {
		t.Error(err)
	}

	err = ds.Delete(struc)
	if err.Error() != errInfo {
		t.Error(err)
	}

	err = ds.Insert(struc)
	if err.Error() != errInfo {
		t.Error(err)
	}

	err = ds.Update(struc)
	if err.Error() != errInfo {
		t.Error(err)
	}

	err = ds.UpdateRow(struc, "")
	if err.Error() != errInfo {
		t.Error(err)
	}

	err = ds.Load(&struc)
	if err.Error() != errInfo {
		t.Error(err)
	}

	err = ds.LoadRow(&struc, "")
	if err.Error() != errInfo {
		t.Error(err)
	}

	se, _ := ds.NewSession()
	err = se.LoadRow(&struc, "")
	if err.Error() != errInfo {
		t.Error(err)
	}

	err = se.UpdateRow(struc, "")
	if err.Error() != errInfo {
		t.Error(err)
	}
}

//测试成功进行所有操作
func Test_CURD_1(t *testing.T) {
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	the_time := time.Date(2014, 1, 7, 5, 50, 4, 0, time.Local)
	bytes := make([]byte, 1024)
	var struc11 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 30, 8.88, true, the_time, bytes}
	var struc12 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}
	var struc13 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 32, 8.88, true, the_time, bytes}
	var struc14 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 33, 8.88, true, the_time, bytes}

	var struc21 userInfoEx = userInfoEx{10, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 40, 8.88, true, the_time, bytes}
	var struc22 userInfoEx = userInfoEx{11, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 41, 8.88, true, the_time, bytes}
	var struc23 userInfoEx = userInfoEx{12, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 42, 8.88, true, the_time, bytes}
	var struc24 userInfoEx = userInfoEx{13, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 43, 8.88, true, the_time, bytes}

	stucslice1 := []UserInfo{struc12, struc13, struc14}
	stucslice2 := []userInfoEx{struc22, struc23, struc24}

	//test exec
	_, err = ds.Exec("delete from userinfo")
	if err != nil {
		t.Error(err)
		return
	}

	//test save
	err = ds.Save(struc11, stucslice1, &struc21, &stucslice2)
	if err != nil {
		t.Error(err)
		return
	}
	cnt, _ := ds.Numerical("select count(*) from userinfo")
	if cnt != 8 {
		t.Error("save err")
		return
	}

	//test load
	tmp2 := userInfoEx{M_id: 10}
	tmpslice2 := []userInfoEx{userInfoEx{M_id: 11}, userInfoEx{M_id: 12}, userInfoEx{M_id: 13}}

	err = ds.Load(&tmp2, &tmpslice2)
	if err != nil {
		t.Error(err)
		return
	}
	if tmp2.M_age != 40 || tmpslice2[0].M_age != 41 || tmpslice2[1].M_age != 42 || tmpslice2[2].M_age != 43 {
		t.Error("load err")
		return
	}

	//test loadRow
	//tmp1 := UserInfo{}
	tmpslice1 := []UserInfo{UserInfo{}, UserInfo{}, UserInfo{}}

	//ds.LoadRow(&tmp1, "where ")
	err = ds.LoadRow(&tmpslice1, "where age < 40 order by age")
	if err != nil {
		t.Error(err)
		return
	}
	if len(tmpslice1) != 4 || tmpslice1[0].M_age != 30 || tmpslice1[3].M_age != 33 {
		t.Error("loadRow err")
	}

	//test update
	tmp2.M_age = 50
	tmpslice2[0].M_age = 51
	tmpslice2[1].M_age = 52
	tmpslice2[2].M_age = 53
	err = ds.Update(&tmpslice2, tmp2)
	if err != nil {
		t.Error(err)
		return
	}
	tmp2.M_age = 0
	tmpslice2[0].M_age = 0
	tmpslice2[1].M_age = 0
	tmpslice2[2].M_age = 0
	err = ds.Load(&tmpslice2, &tmp2)
	if err != nil {
		t.Error(err)
		return
	}
	if tmp2.M_age != 50 || tmpslice2[0].M_age != 51 || tmpslice2[1].M_age != 52 || tmpslice2[2].M_age != 53 {
		t.Error("update err")
		return
	}

	//test updaterow
	cs := cashAndSex{9.99, false}
	err = ds.UpdateRow(cs, "where age > ?", 40)
	if err != nil {
		t.Error(err)
		return
	}
	err = ds.LoadRow(&tmpslice2, "where age > ?", 40)
	if err != nil {
		t.Error(err)
		return
	}
	if len(tmpslice2) != 4 || tmpslice2[0].M_cash != 9.99 || tmpslice2[0].M_sex != false {
		t.Error("updaterow err")
	}

	//test session.updaterow
	se, _ := ds.NewSession()
	cs = cashAndSex{8.88, true}
	err = se.UpdateRow(cs, "where age > ?", 40)
	if err != nil {
		t.Error(err)
		return
	}
	err = se.LoadRow(&tmpslice2, "where age > ? and  用户名 = ? ", 40, "张煜昕")
	if err != nil {
		t.Error(err)
		return
	}
	if len(tmpslice2) != 4 || tmpslice2[0].M_cash != 8.88 || tmpslice2[0].M_sex != true {
		t.Error("se.updaterow err")
	}
	err = se.Commit()
	if err != nil {
		t.Error(err)
		return
	}

	//test delete
	err = ds.Delete(tmpslice2)
	if err != nil {
		t.Error(err)
		return
	}
	cnt, _ = ds.Numerical("select count(*) from userinfo")
	if cnt != 4 {
		t.Error("delete err")
		return
	}

	//test insert
	var struc15 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 34, 8.88, true, the_time, bytes}
	err = ds.Insert(&tmpslice2, struc15)
	if err != nil {
		t.Error(err)
		return
	}
	cnt, _ = ds.Numerical("select count(*) from userinfo")
	if cnt != 9 {
		t.Error("delete err")
		return
	}
}

//测试sturct中字段错误，数据库中没有相应的字段
func Test_filedNameError_1(t *testing.T) {
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
		return
	}

	ds.Register("userinfo", UserInfo_alias{})
	err = ds.Save(UserInfo_alias{})
	if err != nil {
		t.Log(err.Error())
	} else {
		t.Error("不可能吧")
	}
}

//测试错误的数据类型
func Test_structTypeError_1(t *testing.T) {
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	err = ds.Insert(3)
	if err.Error() != "orm: 未找到int对应的table，未使用dataset.SetTable进行绑定或数据类型不正确" {
		t.Error(err)
	}

	err = ds.Save(make(map[int]UserInfo))
	if err.Error() != "orm: 未找到map[int]orm.UserInfo对应的table，未使用dataset.SetTable进行绑定或数据类型不正确" {
		t.Error(err)
	}
}

//测试保存重复的数据
func Test_Save_1(t *testing.T) {
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	the_time := time.Date(2014, 1, 7, 5, 50, 4, 0, time.Local)
	bytes := make([]byte, 1024)
	var struc11 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}
	var struc12 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}

	var struc21 userInfoEx = userInfoEx{11, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 40, 8.88, true, the_time, bytes}
	var struc22 userInfoEx = userInfoEx{11, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 41, 8.88, true, the_time, bytes}

	_, err = ds.Exec("delete from userinfo")
	if err != nil {
		t.Error(err)
		return
	}

	err = ds.Save(struc11, struc12, &struc21, &struc22)
	if err != nil {
		t.Error(err)
		return
	}
	cnt, _ := ds.Numerical("select count(*) from userinfo")
	if cnt != 2 {
		t.Error("save err")
		return
	}
}

//测试插入重复的数据
func Test_InsertError_1(t *testing.T) {
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	the_time := time.Date(2014, 1, 7, 5, 50, 4, 0, time.Local)
	bytes := make([]byte, 1024)
	var struc11 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}
	var struc12 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}

	_, err = ds.Exec("delete from userinfo")
	if err != nil {
		t.Error(err)
		return
	}

	err = ds.Insert(struc11, struc12)
	if err.Error() != "UNIQUE constraint failed: userinfo.age" {
		t.Error(err)
	}
}

//测试Update无主键数据
func Test_UpdateError_1(t *testing.T) {
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	the_time := time.Date(2014, 1, 7, 5, 50, 4, 0, time.Local)
	bytes := make([]byte, 1024)
	var struc11 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}
	var struc12 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}

	err = ds.Update(&struc11, struc12)
	if err.Error() != "orm: 结构体orm.UserInfo中未找到主键字段" {
		t.Error(err)
	}
}

//测试UpdateRow参数不正确
func Test_UpdateRowError_1(t *testing.T) {
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	the_time := time.Date(2014, 1, 7, 5, 50, 4, 0, time.Local)
	bytes := make([]byte, 1024)
	var struc11 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}
	var struc12 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}

	slice := []UserInfo{struc11, struc12}
	err = ds.UpdateRow(slice, "")
	if err.Error() != "orm: 输入参数的类型只能为Struct，实际为slice" {
		t.Error(err)
	}
}

//测试Load，loadRow条件不满足
func Test_LoadError_1(t *testing.T) {
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	the_time := time.Date(2014, 1, 7, 5, 50, 4, 0, time.Local)
	bytes := make([]byte, 1024)
	var struc11 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 30, 8.88, true, the_time, bytes}
	var struc12 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}
	var struc21 userInfoEx = userInfoEx{111, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 40, 8.88, true, the_time, bytes}
	ds.Save(struc11, struc12)

	slice := []userInfoEx{struc21, struc21}
	err = ds.Load(&struc21)
	if err != sql.ErrNoRows {
		t.Error(err)
	}

	err = ds.Load(&slice)
	if err != sql.ErrNoRows {
		t.Error(err)
	}

	err = ds.LoadRow(&slice, "where age > ?", 100)
	if err != nil || len(slice) != 0 {
		t.Error("不应该啊", err, len(slice))
	}
}

//测试UpdateRow没有记录满足条件
func Test_UpdateRowError_2(t *testing.T) {
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		t.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	the_time := time.Date(2014, 1, 7, 5, 50, 4, 0, time.Local)
	bytes := make([]byte, 1024)
	var struc11 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 30, 8.88, true, the_time, bytes}
	var struc12 UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 31, 8.88, true, the_time, bytes}
	var struc21 userInfoEx = userInfoEx{111, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 40, 8.88, true, the_time, bytes}

	err = ds.Save(struc11, struc12)
	if err != nil {
		t.Error(err)
		return
	}

	err = ds.UpdateRow(struc21, "where age < ?", 10)
	if err != sql.ErrNoRows {
		t.Error(err)
	}

	err = ds.Update(struc21)
	if err != sql.ErrNoRows {
		t.Error(err)
	}
}

//测试单行写入效率
func Benchmark_SaveOne(b *testing.B) {
	b.StopTimer()
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		b.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})
	_, err = ds.Exec("delete from userinfo")
	if err != nil {
		b.Error(err)
		return
	}
	the_time := time.Date(2014, 1, 7, 5, 50, 4, 0, time.Local)
	bytes := make([]byte, 1024)
	var struc UserInfo = UserInfo{0, 0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 30, 8.88, true, the_time, bytes}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := ds.Save(struc)
		if err != nil {
			b.Log(err.Error())
		}
		struc.M_age++
	}
}

//测试多行写入效率
func Benchmark_Save1000(b *testing.B) {
	b.StopTimer()
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		b.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})
	_, err = ds.Exec("delete from userinfo")
	if err != nil {
		b.Error(err)
		return
	}
	the_time := time.Date(2014, 1, 7, 5, 50, 4, 0, time.Local)
	bytes := make([]byte, 1024)
	var struc userInfoEx = userInfoEx{0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 30, 8.88, true, the_time, bytes}

	tmp := make([]userInfoEx, 1000)
	for i := 0; i < 1000; i++ {
		tmp[i] = struc
		struc.M_age++
		struc.M_id++
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := ds.Save(tmp)
		if err != nil {
			b.Log(err.Error())
		}
	}

}

//测试读取单行效率
func Benchmark_LoadOne(b *testing.B) {
	b.StopTimer()
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		b.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	var struc userInfoEx = userInfoEx{M_id: 55}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := ds.Load(&struc)
		if err != nil {
			b.Log(err.Error())
		}
	}

}

//测试读取单行效率
func Benchmark_LoadRowOne(b *testing.B) {
	b.StopTimer()
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		b.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	var struc userInfoEx = userInfoEx{M_id: 55}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := ds.LoadRow(&struc, "where uid = 55")
		if err != nil {
			b.Log(err.Error())
		}
	}

}

//测试读取多行效率,一次读1000行
func Benchmark_Load1000(b *testing.B) {
	b.StopTimer()
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		b.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	slice := make([]userInfoEx, 1000)
	for j, v := range slice {
		v.M_id = j
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := ds.Load(&slice)
		if err != nil {
			b.Log(err.Error())
		}
		if len(slice) != 1000 {
			b.Error(len(slice))
		}
	}
}

//测试读取多行效率,一次读1000行
func Benchmark_LoadRow1000(b *testing.B) {
	b.StopTimer()
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		b.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	slice := make([]userInfoEx, 0)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := ds.LoadRow(&slice, "limit 1000")
		if err != nil {
			b.Log(err.Error())
		}
		if len(slice) != 1000 {
			b.Error(len(slice))
		}
	}
}

//测试单行写入效率
func Benchmark_SqlSaveOne(b *testing.B) {
	b.StopTimer()
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		b.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})
	_, err = ds.Exec("delete from userinfo")
	if err != nil {
		b.Error(err)
		return
	}
	the_time := time.Date(2014, 1, 7, 5, 50, 4, 0, time.Local)
	bytes := make([]byte, 1024)
	var struc userInfoEx = userInfoEx{0, "张煜昕", "银河系太阳系地球亚洲中国湖北省潜江市辉煌村八组", 30, 8.88, true, the_time, bytes}

	tb, err := ds.findTable(reflect.TypeOf(struc))
	if err != nil {
		b.Error(err)
		return
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := ds.Exec(tb.replaceStr, struc.M_id, struc.M_username, struc.M_city, struc.M_age, struc.M_cash, struc.M_sex, struc.M_timestamp, struc.M_zip)
		if err != nil {
			b.Log(err.Error())
		}
		struc.M_age++
		struc.M_id++
	}
}

//测试单行读取效率,使用最直接方式
func Benchmark_SqlLoadOne(b *testing.B) {
	b.StopTimer()
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		b.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	var struc userInfoEx

	tb, err := ds.findTable(reflect.TypeOf(struc))
	if err != nil {
		b.Error(err)
		return
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		qe := tb.selectStr + "where uid = 55"
		row := ds.QueryRow(qe)
		err = row.Scan(&struc.M_id, &struc.M_username, &struc.M_city, &struc.M_age, &struc.M_cash, &struc.M_sex, &struc.M_timestamp, &struc.M_zip)
		if err != nil {
			b.Error(err)
		}
	}
}

//测试单行读取效率,使用key2if
func Benchmark_SLO_key2IfAddr(b *testing.B) {
	b.StopTimer()
	ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
	if err != nil {
		b.Error(err)
		return
	}
	ds.Register("userinfo", userInfoEx{})
	ds.Register("userinfo", UserInfo{})
	ds.Register("userinfo", cashAndSex{})

	var struc userInfoEx

	tb, err := ds.findTable(reflect.TypeOf(struc))
	if err != nil {
		b.Error(err)
		return
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		qe := tb.selectStr + "where uid = 55"
		row := ds.QueryRow(qe)
		err = row.Scan(tb.key2IfAddr(reflect.ValueOf(&struc).Elem())...)
		if err != nil {
			b.Error(err)
		}
	}
}

//测试单行读取效率,每次findtable
func Benchmark_SLO_findTable(b *testing.B) {
	var struc userInfoEx

	for i := 0; i < b.N; i++ {
		ds, err := NewDataSet("sqlite3", "./db_testSqlite3")
		if err != nil {
			b.Error(err)
			return
		}
		ds.Register("userinfo", userInfoEx{})
		ds.Register("userinfo", UserInfo{})
		ds.Register("userinfo", cashAndSex{})

		tb, err := ds.findTable(reflect.TypeOf(struc))
		if err != nil {
			b.Error(err)
			return
		}

		qe := tb.selectStr + "where uid = 55"
		row := ds.QueryRow(qe)
		err = row.Scan(tb.key2IfAddr(reflect.ValueOf(&struc).Elem())...)
		if err != nil {
			b.Error(err)
		}
	}
}
