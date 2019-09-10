package graphorm

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCombineOrders(t *testing.T) {
	require.Equal(t, JoinOrders([]string{"name"}), "name asc")
	require.Equal(t, JoinOrders([]string{"name", "-", "-password"}), "name asc,password desc")
}

type Corp struct {
	Model
	Name      string
	ManagerID UUID
	Manager   Manager `gorm:"ForeignKey:ManagerID"`
}

type Manager struct {
	Model
	Name string
}

func TestGorm(t *testing.T) {
	var db *gorm.DB
	// connect to the example db, create if it doesn't exist
	db, err := gorm.Open("sqlite3", "./test.sqlite3")
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	db.DropTableIfExists(&Corp{}, &Manager{})
	db.AutoMigrate(&Corp{}, &Manager{})

	t.Run("Test Save Association", func(t *testing.T) {
		corp := Corp{
			Name:    "apple",
			Manager: Manager{Name: "zhang3"},
		}

		require.NoError(t, db.Save(&corp).Error)

		c2 := Corp{}
		err = db.First(&c2, "id = ?", corp.ID).Related(&c2.Manager).Error
		require.NoError(t, err)
		require.Equal(t, "zhang3", c2.Manager.Name)
	})

	t.Run("Test Cnt", func(t *testing.T) {
		corp := Corp{
			Name:    "apple",
			Manager: Manager{Name: "zhang3"},
		}

		require.NoError(t, db.Save(&corp).Error)

		cnt := 0
		err := db.Model(&Corp{}).Scopes(Like("name", "a")).Count(&cnt).Error
		require.NoError(t, err)
		require.Equal(t, 1, cnt)

		err = db.Model(&Corp{}).Scopes(Like("name", "xxxx")).Count(&cnt).Error
		require.NoError(t, err)
		require.Equal(t, 0, cnt)
	})

	t.Run("Test Delete Association", func(t *testing.T) {
		corp := Corp{
			Name:    "banana",
			Manager: Manager{Name: "li4"},
		}
		require.NoError(t, db.Save(&corp).Error)

		err = db.Delete(&corp).Error
		require.NoError(t, err)

		c2 := Corp{}
		err = db.First(&c2, "id = ?", corp.ID).Error
		require.EqualError(t, err, "record not found")

		err = db.Model(&corp).Related(&c2.Manager).Error
		require.NoError(t, err)
		require.Equal(t, "li4", c2.Manager.Name)

		err = db.First(&c2.Manager, "id = ?", corp.ManagerID).Error
		require.NoError(t, err)
		require.Equal(t, "li4", c2.Manager.Name)

		//delete again
		corp.ID = NewUUID()
		err = db.Delete(&corp).Error
		require.NoError(t, err)
	})
}
