// dataset project doc.go

/*
dataset document
	用法：
	1）一个数据库对象使用一个DataSet对象即可，可以在多个协程中使用。
		ds, err := orm.NewDataSet("sqlite3", "./db_testSqlite3")

	2）设置要操作的表格以及对应的结构体类型，该操作是非线程安全的，最好在程序初始化时设置完毕。
		结构体的定义如下(不支持嵌套)：
		type RAE struc {
			Id    int64 `PK:"uid"` //PK表示主键，字段名为"uid"
			R     int   `距离`//对应表格中的字段“距离”
			A     int   `方位` //同上
			E     int    `俯仰` //同上
		}
		结构体名可以小写，字段名必须大写开头。

		//"userinfo"为表名
		ds.SetTable("userinfo", userInfoEx{})

	3）使用Insert，Update，Save, Delete等操作进行増删改查；如果需要进行多个事务操作，使用NewSession创建一个Session
		对象来进行操作，接口与dataset基本一致。

	完整示例参见sqlite3_test.go中的函数Test_CURD_1

*/
package orm
