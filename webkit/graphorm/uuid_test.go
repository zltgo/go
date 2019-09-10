package graphorm

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/require"
)

type User struct {
	Model
	Name         string `gorm:"unique_index"`
	PasswordHash MD5
}

type usrInDB struct {
	ID           string `sql:"type:uuid;primary_key"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time `sql:"index"`
	PasswordHash string
}

func TestUUID(t *testing.T) {
	t.Run("Test invalid UUID", func(t *testing.T) {
		require.Equal(t, UUID("123").IsValid(), false)
		require.Equal(t, UUID("5d318457e1382359d0896434").IsValid(), true)
		require.Equal(t, UUID("5d318457e1382359d08964g4").IsValid(), false)
	})

	t.Run("Test zero UUID", func(t *testing.T) {
		require.Equal(t, 12, len(string([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})))
		require.Equal(t, ZeroUUID().IsValid(), true)
	})

	t.Run("Test Scan", func(t *testing.T) {
		var u UUID
		u.Scan("aaaaaaaaaaaa")
		require.Equal(t, string(u), "616161616161616161616161")

		u.Scan([]byte("aaaaaaaaaaaa"))
		require.Equal(t, string(u), "616161616161616161616161")

		u.Scan("bbbbbbbbbbbb")
		require.Equal(t, string(u), "626262626262626262626262")

		require.Equal(t, u.Scan("123").Error(), "invalid UUID")
		require.Equal(t, u.Scan(456).Error(), "cannot convert int to UUID")
	})

	t.Run("Test UnmarshalGQL", func(t *testing.T) {
		var u UUID
		u.UnmarshalGQL("616161616161616161616161")
		require.Equal(t, string(u), "616161616161616161616161")

		u.UnmarshalGQL("626262626262626262626262")
		require.Equal(t, string(u), "626262626262626262626262")

		require.Equal(t, u.UnmarshalGQL("123").Error(), "invalid UUID: 123")
		require.Equal(t, u.UnmarshalGQL(456).Error(), "cannot convert int to UUID")
	})
}

func TestUUIDAndMD5WithDB(t *testing.T) {
	var db *gorm.DB
	// connect to the example db, create if it doesn't exist
	db, err := gorm.Open("sqlite3", "./test.sqlite3")
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	db.DropTableIfExists(&User{})
	db.AutoMigrate(&User{})

	t.Run("Test Create ID manually", func(t *testing.T) {
		z3 := User{
			Model{ID: "616161616161616161616161"},
			"zhang3",
			"62626262626262626262626262626262",
		}

		require.Equal(t, db.Save(&z3).Error, nil)
		z3.Name = "aaa"
		require.Equal(t, 409, ErrorCode(db.Create(&z3).Error))

		u := User{}
		err := db.Where("name = ?", "zhang3").First(&u).Error
		require.Equal(t, err, nil)
		require.Equal(t, u.Name, "zhang3")
		require.Equal(t, u.ID.String(), "616161616161616161616161")
		require.Equal(t, u.PasswordHash.String(), "62626262626262626262626262626262")

		u2 := usrInDB{}
		err = db.Table("users").Where("name = ?", "zhang3").First(&u2).Error
		require.Equal(t, u2.ID, "aaaaaaaaaaaa")
		require.Equal(t, u2.PasswordHash, "bbbbbbbbbbbbbbbb")
	})

	t.Run("Test Create ID automatically", func(t *testing.T) {
		u1 := User{Name: "li4", PasswordHash: "62626262626262626262626262626262"}
		require.Equal(t, db.Create(&u1).Error, nil)

		//duplicate name
		u11 := User{Name: "li4", PasswordHash: "62626262626262626262626262626262"}
		require.Equal(t, ErrorCode(db.Create(&u11).Error), 409)

		u2 := User{}
		err := db.Where("name = ?", "li4").First(&u2).Error
		require.Equal(t, err, nil)
		require.Equal(t, u2.Name, "li4")
		require.Equal(t, u2.ID.IsValid(), true)
		require.Equal(t, u2.PasswordHash.IsValid(), true)
	})

	t.Run("Test Get User by UUID", func(t *testing.T) {
		u1 := User{Name: "wang5", PasswordHash: "62626262626262626262626262626262"}
		require.Equal(t, db.Create(&u1).Error, nil)

		u2 := User{}
		err := db.Scopes(Like("name", "wang")).Debug().First(&u2).Error
		require.Equal(t, err, nil)
		require.Equal(t, u2.ID.IsValid(), true)
		require.Equal(t, u2.Name, "wang5")

		u3 := User{}
		err = db.First(&u3, "id = ?", u2.ID).Error
		require.Equal(t, err, nil)
		require.Equal(t, u3.Name, "wang5")

		//users := make([]User, 0)
		//err = db.Debug().Order("name; drop table 'users'").Find(&users).Error
		//require.Equal(t, err,  nil)
	})

	t.Run("Test time compare without time parse", func(t *testing.T) {
		u1 := User{Name: "chen6", PasswordHash: "62626262626262626262626262626262"}
		require.Equal(t, db.Create(&u1).Error, nil)
		u1.CreatedAt = u1.CreatedAt.Add(-time.Nanosecond)

		u2 := User{}
		err := db.Where("created_at > ?", u1.CreatedAt).First(&u2).Error
		t.Log(err, u2.Name)
	})

	t.Run("Test time compare with time parse", func(t *testing.T) {
		myDb, err := gorm.Open("sqlite3", "./test.sqlite3?charset=utf8&parseTime=True&loc=Local")
		if err != nil {
			t.Error(err)
			return
		}
		defer myDb.Close()

		u1 := User{Name: "guan7", PasswordHash: "62626262626262626262626262626262"}
		require.Equal(t, myDb.Create(&u1).Error, nil)
		min := u1.CreatedAt.Add(-time.Nanosecond)
		max := u1.CreatedAt.Add(time.Second)

		u2 := User{}
		err = myDb.Debug().Where("created_at > ?", &min).Where("created_at < ?", &max).First(&u2).Error
		require.Equal(t, err, nil)
		require.Equal(t, u2.Name, "guan7")

		u3 := User{}
		err = db.Scopes(Between("created_at", []time.Time{min, max})).Debug().First(&u3).Error
		require.Equal(t, err, nil)
		require.Equal(t, u3.Name, "guan7")

		u4 := User{}
		err = db.Scopes(Between("created_at", []*time.Time{&min, nil})).Debug().First(&u4).Error
		require.Equal(t, err, nil)
		require.Equal(t, u4.Name, "guan7")
	})

	t.Run("Test deletedAt", func(t *testing.T) {
		u := User{Name: "lu8", PasswordHash: "62626262626262626262626262626262"}
		require.Equal(t, db.Create(&u).Error, nil)

		err := db.Where("name = ?", "lu8").Delete(&User{}).Error
		require.NoError(t, err)

		u1 := User{}
		err = db.Scopes(Unscoped(0)).Where("name = ?", "lu8").Debug().First(&u1).Error
		require.Errorf(t, err, "record not found")

		u3 := User{}
		err = db.Scopes(Unscoped(1)).Debug().Where("name = ?", "lu8").First(&u3).Error
		require.NoError(t, err)
		require.Equal(t, u3.Name, "lu8")

		u2 := User{}
		err = db.Scopes(Unscoped(2)).Debug().First(&u2).Error
		require.NoError(t, err)
		require.Equal(t, u2.Name, "lu8")
	})

	t.Run("Test Count", func(t *testing.T) {
		cnt := 0
		err := db.Model(&User{}).Where("id = ?", "11111").Count(&cnt).Error
		require.NoError(t, err)
	})

	t.Run("Test Update", func(t *testing.T) {
		u1 := User{Name: "dai9", PasswordHash: "62626262626262626262626262626262"}
		require.Equal(t, db.Create(&u1).Error, nil)

		u2 := struct {
			ID           UUID
			Name         string `gorm:"unique_index"`
			PasswordHash MD5
		}{u1.ID, "dai99", "62626262626262626262626262626262"}
		require.NoError(t, db.Debug().Model(&User{}).Update(&u2).Error)

		us := make([]User, 0)
		err = db.Scopes(Like("name", "dai")).Find(&us).Error
		require.NoError(t, err)
		require.Len(t, us, 1)
		require.Equal(t, "dai99", us[0].Name)
	})
}
