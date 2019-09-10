package graphorm

import (
	"github.com/jinzhu/gorm"
	"reflect"
)

// shortcut of JoinOrders and db.Order
func OrderBy(orders []string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if joined := JoinOrders(orders); joined != "" {
			return db.Order(joined)
		}
		return db
	}
}

// query records between ranged conditions
// ranges must has be a slice like []int{1,3} or []*int{nil, &3}
func Between(column string, ranges interface{}) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if ranges == nil {
			return db
		}
		//get elements of slice
		vs := reflect.ValueOf(ranges)
		switch vs.Len() {
		case 1:
			//only one parameter means equal
			db = db.Where(column+" = ?", vs.Index(0).Interface())
		case 2:
			if left := reflect.Indirect(vs.Index(0)); left.IsValid() {
				db = db.Where(column+" >= ?", left.Interface())
			}
			if right := reflect.Indirect(vs.Index(1)); right.IsValid() {
				db = db.Where(column+" <= ?", right.Interface())
			}
		}
		return db
	}
}

// like query
func Like(column string, key string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if key != "" {
			return db.Where(column+" like ?", "%"+key+"%")
		}
		return db
	}
}

// flag can be bool or int type.
// flag 0: return records excluding deleted record.
// flag 1: return all record including deleted record.
// flag 2: return only deleted records.
func Unscoped(flag interface{}) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		var i int64 = 0
		v := reflect.ValueOf(flag)
		if reflect.TypeOf(flag).Kind() == reflect.Bool {
			if v.Bool() {
				i = 1
			}
		} else { //reflect.Int
			i = v.Int()
		}

		switch i {
		case 1:
			return db.Unscoped()
			//不能用<>null或者 =null 来对null进行判断，因为null什么也不是
			//只能用is null 或者 is not null
		case 2:
			return db.Unscoped().Where("deleted_at is not null")
		default: //case 0
			return db
		}
	}
}

//query records after id order by id asc
func AfterID(id *UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if id != nil {
			return db.Where("id > ?", *id)
		}
		return db
	}
}

// prefix '-' means desc, otherwise asc.
func JoinOrders(orders []string) (joined string) {
	for _, order := range orders {
		if order != "" {
			if order[0] == '-' {
				if len(order) > 1 {
					joined += "," + order[1:] + " desc"
				}
			} else {
				joined += "," + order + " asc"
			}
		}
	}
	//trim prefix ','
	if joined != "" {
		joined = joined[1:]
	}
	return joined
}
