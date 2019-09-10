package model

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/zltgo/webkit/ginx"
	gm "github.com/zltgo/webkit/graphorm"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func timeFunc(sec int64, nsec int64) func() time.Time {
	return func() time.Time {
		return time.Unix(sec, nsec)
	}
}

var (
	m  *Model
	af = &AuthInfo{
		Empno: "root",
		Role:  AdminRole,
	}
	emptyUUID = gm.UUID("")
	zeroUUID  = gm.ZeroUUID()
)

func init() {
	//delete db file
	err := os.Remove("./test.sqlite3")
	if err != nil {
		log.Println(err)
	}

	m, err = NewModel(Options{
		Source: "./test.sqlite3?charset=utf8&parseTime=True&loc=Local",
	})
	if err != nil {
		log.Fatal(err)
	}

	//add some records
	addAccount("马云", "15932323232", "淘宝tmall")
	addAccount("马化腾", "15987653275", "微信wechat")
	addAccount("李彦宏", "18609837575", "百度baidu")
}

func addAccount(realname, mobile, corp string) gm.UUID {
	af := &AuthInfo{
		Empno: "root",
		Role:  AdminRole,
	}

	ac, code, err := m.CreateAccount(af, &NewAccount{
		Empno:            "000",
		Password:         "000000000",
		RealName:         realname,
		Mobile:           mobile,
		Corp:             corp,
		MaxStores:        10,
		MaxUsersPerStore: 100,
		ExpiresAt:        time.Now().Add(time.Hour * 24),
		Remarks:          fmt.Sprintf("公司：%s, 手机：%s", corp, mobile),
	})
	if err != nil {
		log.Println(code, err)
		return zeroUUID
	}
	addStoreAddUser(ac.ID, "分店0", "0")
	addStoreAddUser(ac.ID, "分店1", "1")
	addStoreAddUser(ac.ID, "分店2", "2")

	return ac.ID
}

func addStoreAddUser(accountId gm.UUID, name string, empnoPrefix string) {
	st, code, err := m.CreateStore(af, &NewStore{
		AccountID: accountId,
		Name:      name,
		Remarks:   fmt.Sprintf("分店名：%s", name),
	})
	if err != nil {
		log.Println(code, err)
		return
	}

	//add employee
	addUser(accountId, st.ID, empnoPrefix+"001", "000000001", "张0", "级别0")
	addUser(accountId, st.ID, empnoPrefix+"002", "000000002", "张1", "级别1")
	addUser(accountId, st.ID, empnoPrefix+"003", "000000003", "张2", "级别2")
}

func addUser(accountId, storeId gm.UUID, empno, password, realName, role string) {
	_, code, err := m.CreateUser(af, &NewUser{
		AccountID: accountId,
		Empno:     empno,
		Password:  password,
		RealName:  realName,
		Role:      role,
		StoreID:   storeId,
	})
	if err != nil {
		log.Println(code, err)
	}
}

func TestToken(t *testing.T) {
	t.Run("Admin", func(t *testing.T) {
		af, code, err := m.Admin("root", "112358")
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Equal(t, "root", af.Empno)
		require.Equal(t, m.opts.Admin.PasswordHash, af.PasswordHash)
		require.Equal(t, AdminRole, af.Role)
		require.Greater(t, af.ExpiresAt.Unix(), time.Now().Unix())

		af, code, err = m.Admin("root", "wrong password")
		require.EqualError(t, err, "name or password error")
		require.Equal(t, 401, code)
	})

	t.Run("GetAccountIdByMobile", func(t *testing.T) {
		_, code, err := m.GetAccountIdByMobile("18609837575")
		require.NoError(t, err)
		require.Equal(t, 200, code)

		_, code, err = m.GetAccountIdByMobile("18609837576")
		require.EqualError(t, err, "record not found")
		require.Equal(t, 404, code)

		_, code, err = m.GetAccountIdByMobile("invalid number")
		require.Equal(t, true, ginx.IsValidationError(err))
		require.Equal(t, 400, code)
	})

	t.Run("authToken and refreshToken", func(t *testing.T) {
		id, code, err := m.GetAccountIdByMobile("18609837575")
		require.NoError(t, err)
		require.Equal(t, 200, code)

		af, code, err := m.Login(&AuthForm{
			AccountID: id,
			Empno:     "000",
			Password:  "000000000",
		})
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Equal(t, id, af.AccountID)
		require.Equal(t, "000", af.Empno)
		require.Equal(t, ManagerRole, af.Role)

		rf, code, err := m.RefreshToken(af)
		require.NoError(t, err)
		require.Equal(t, rf, af)
	})

	t.Run("expires at", func(t *testing.T) {
		ac, _, err := m.CreateAccount(af, &NewAccount{
			Empno:     "000",
			Password:  "000000000",
			RealName:  "测试ExpiresAt",
			Mobile:    "13500000000",
			Corp:      "测试用",
			ExpiresAt: time.Now().Add(-time.Hour),
		})
		defer m.DeleteAccount(af, ac.ID)
		require.NoError(t, err)

		ac2, _, err := m.GetAccountByID(af, ac.ID)
		require.NoError(t, err)
		require.Equal(t, ac.ExpiresAt.Format("20060102T150405Z"), ac2.ExpiresAt.Format("20060102T150405Z"))

		_, code, err := m.Login(&AuthForm{
			AccountID: ac.ID,
			Empno:     "000",
			Password:  "000000000",
		})
		require.Equal(t, code, http.StatusPaymentRequired)
		require.Error(t, err)
		require.True(t, strings.HasPrefix(err.Error(), "account expired"))
	})
}

func TestCreate(t *testing.T) {
	newAc := &NewAccount{
		Empno:            "000",
		Password:         "000000000",
		RealName:         "测试一下",
		Mobile:           "13500000001",
		Corp:             "测试CreateAccount",
		MaxStores:        2,
		MaxUsersPerStore: 2,
		ExpiresAt:        time.Now().Add(-time.Hour),
	}
	defer func() {
		id, _, _ := m.GetAccountIdByMobile("13500000001")
		m.DeleteAccount(af, id)
	}()

	t.Run("CreateAccount", func(t *testing.T) {
		_, code, err := m.CreateAccount(af, newAc)
		require.NoError(t, err)
		require.Equal(t, 200, code)

		//409
		newAc.Mobile = "18609837575"
		_, code, err = m.GetAccountByID(&AuthInfo{
			Empno: "root",
			Role:  AdminRole,
		}, zeroUUID)
		_, code, err = m.CreateAccount(af, newAc)
		require.EqualError(t, err, "UNIQUE constraint failed: accounts.mobile")
		require.Equal(t, 409, code)

		//400
		newAc.Empno = "empnotoolang"
		_, code, err = m.CreateAccount(af, newAc)
		require.True(t, ginx.IsValidationError(err))
		require.Equal(t, 400, code)
	})

	mySt := &Store{}
	t.Run("CreateStore", func(t *testing.T) {
		id, _, err := m.GetAccountIdByMobile("13500000001")
		require.NoError(t, err)

		st, code, err := m.CreateStore(af, &NewStore{
			AccountID: id,
			Name:      "测试一下创建分店",
		})
		require.NoError(t, err)
		require.Equal(t, 200, code)
		mySt = st

		//409
		_, code, err = m.CreateStore(af, &NewStore{
			AccountID: id,
			Name:      "测试一下创建分店",
		})
		require.EqualError(t, err, "UNIQUE constraint failed: stores.account_id, stores.name")
		require.Equal(t, 409, code)

		//412
		_, _, err = m.CreateStore(af, &NewStore{
			AccountID: id,
			Name:      "再创建一个分店",
		})
		require.NoError(t, err)
		_, code, err = m.CreateStore(af, &NewStore{
			AccountID: id,
			Name:      "创建分店过多",
		})
		require.Equal(t, 412, code)
		require.True(t, strings.HasSuffix(err.Error(), "创建的分店数量已达上限：2"))

		//400
		_, code, err = m.CreateStore(af, &NewStore{
			AccountID: id,
			Name:      ";/-无效的名字",
		})
		require.True(t, ginx.IsValidationError(err))
		require.Equal(t, 400, code)
	})

	t.Run("CreateUser", func(t *testing.T) {
		id, _, err := m.GetAccountIdByMobile("13500000001")
		require.NoError(t, err)

		newUsr := &NewUser{
			AccountID: id,
			Empno:     "003",
			Password:  "000000003",
			RealName:  "测试创建用户",
			Role:      "店长",
			StoreID:   mySt.ID,
		}

		//200
		_, code, err := m.CreateUser(af, newUsr)
		require.NoError(t, err)
		require.Equal(t, 200, code)

		//409
		_, code, err = m.CreateUser(af, newUsr)
		require.EqualError(t, err, "UNIQUE constraint failed: users.account_id, users.empno")
		require.Equal(t, 409, code)

		//412
		newUsr.Empno = "004"
		_, _, err = m.CreateUser(af, newUsr)
		require.NoError(t, err)
		newUsr.Empno = "005"
		_, code, err = m.CreateUser(af, newUsr)
		require.Equal(t, 412, code)
		require.True(t, strings.HasSuffix(err.Error(), "员工数量已达上限：2"))

		//400
		newUsr.Empno = "无效的工号"
		_, code, err = m.CreateUser(af, newUsr)
		require.True(t, ginx.IsValidationError(err))
		require.Equal(t, 400, code)

		newUsr.Empno = "006"
		newUsr.Role = ManagerRole
		_, code, err = m.CreateUser(af, newUsr)
		require.EqualError(t, err, "普通用户不能设置为管理员权限")
		require.Equal(t, 400, code)

		//409
		newUsr.StoreID = gm.NewUUID()
		newUsr.Role = "店长"
		_, code, err = m.CreateUser(af, newUsr)
		require.EqualError(t, err, "FOREIGN KEY constraint failed")
		require.Equal(t, 409, code)
	})
}

func TestUpdate(t *testing.T) {
	addAccount("丁磊", "15932325678", "网易wy")
	//get account
	id, _, err := m.GetAccountIdByMobile("15932325678")
	require.NoError(t, err)
	defer m.DeleteAccount(af, id)

	ac, _, err := m.GetAccountByID(af, id)
	require.NoError(t, err)

	manager, _, err := m.GetManagerOfAccount(af, ac)
	require.NoError(t, err)

	//get stores
	stores, _, err := m.GetStoresOfAccount(af, ac, false)
	require.NoError(t, err)
	require.Len(t, stores, 3)

	t.Run("UpdateAccount", func(t *testing.T) {
		modAc := &ModAccount{
			ID:               id,
			Mobile:           "15932325679",
			Corp:             "新网易wy",
			MaxStores:        11,
			MaxUsersPerStore: 101,
			ExpiresAt:        time.Now().Add(time.Hour),
			Remarks:          "测试改备注",
		}

		code, err := m.UpdateAccount(af, modAc)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		updatedAc, _, err := m.GetAccountByID(af, id)
		require.NoError(t, err)
		require.Equal(t, modAc.Mobile, updatedAc.Mobile)
		require.Equal(t, modAc.Corp, updatedAc.Corp)
		require.Equal(t, modAc.MaxStores, updatedAc.MaxStores)
		require.Equal(t, modAc.MaxUsersPerStore, updatedAc.MaxUsersPerStore)
		require.Equal(t, modAc.ExpiresAt.Unix(), updatedAc.ExpiresAt.Unix())
		require.Equal(t, modAc.Remarks, updatedAc.Remarks)

		//409
		modAc.Mobile = "18609837575"
		code, err = m.UpdateAccount(af, modAc)
		require.EqualError(t, err, "UNIQUE constraint failed: accounts.mobile")
		require.Equal(t, 409, code)

		//404
		modAc.ID = gm.NewUUID()
		code, err = m.UpdateAccount(af, modAc)
		require.EqualError(t, err, "record not found")
		require.Equal(t, 404, code)

		//400
		modAc.Mobile = "invalidMobile"
		code, err = m.UpdateAccount(af, modAc)
		require.True(t, ginx.IsValidationError(err))
		require.Equal(t, 400, code)
	})

	t.Run("UpdateStore", func(t *testing.T) {
		modSt := &ModStore{
			ID:      stores[0].ID,
			Name:    "总店",
			Remarks: "测试更新备注",
		}
		code, err := m.UpdateStore(af, modSt)
		require.NoError(t, err)
		require.Equal(t, 200, code)

		//409
		modSt.Name = "分店2"
		code, err = m.UpdateStore(af, modSt)
		require.EqualError(t, err, "UNIQUE constraint failed: stores.account_id, stores.name")
		require.Equal(t, 409, code)

		//400
		modSt.Name = "_(不是有效的名字)"
		code, err = m.UpdateStore(af, modSt)
		require.True(t, ginx.IsValidationError(err))
		require.Equal(t, 400, code)

		//404
		modSt.ID = zeroUUID
		modSt.Name = "总店"
		code, err = m.UpdateStore(af, modSt)
		require.EqualError(t, err, "record not found")
		require.Equal(t, 404, code)
	})

	t.Run("UpdateUser", func(t *testing.T) {
		users, _, err := m.GetUsersOfStore(af, stores[0], false)
		require.NoError(t, err)
		require.Greater(t, len(users), 1)

		modUsr := &ModUser{
			ID:       users[0].ID,
			Empno:    "0008",
			RealName: "lu8",
			Role:     "总监",
			StoreID:  stores[1].ID,
		}

		code, err := m.UpdateUser(af, modUsr)
		require.NoError(t, err)
		require.Equal(t, 200, code)

		//login
		_, _, err = m.Login(&AuthForm{
			AccountID: id,
			Empno:     "0008",
			Password:  "000000001",
		})
		require.NoError(t, err)

		//400
		modUsr.Role = AdminRole
		code, err = m.UpdateUser(af, modUsr)
		require.EqualError(t, err, "普通用户不能设置为管理员权限")
		require.Equal(t, 400, code)

		//409
		modUsr.Empno = "0002"
		modUsr.Role = "路人甲"
		code, err = m.UpdateUser(af, modUsr)
		require.EqualError(t, err, "UNIQUE constraint failed: users.account_id, users.empno")
		require.Equal(t, 409, code)

		modUsr.Empno = "0009"
		modUsr.StoreID = zeroUUID
		code, err = m.UpdateUser(af, modUsr)
		require.EqualError(t, err, "FOREIGN KEY constraint failed")
		require.Equal(t, 409, code)

		//400
		modUsr.Empno = "_(不是有效的工号)"
		code, err = m.UpdateUser(af, modUsr)
		require.True(t, ginx.IsValidationError(err))
		require.Equal(t, 400, code)

		//404
		modUsr.ID = zeroUUID
		modUsr.Empno = "0009"
		code, err = m.UpdateUser(af, modUsr)
		require.EqualError(t, err, "record not found")
		require.Equal(t, 404, code)
	})

	t.Run("UpdateManager", func(t *testing.T) {
		modMng := &ModManager{
			ID:       manager.ID,
			Empno:    "100",
			RealName: "丁三石",
			Password: "100000000",
		}

		code, err := m.UpdateManager(af, modMng)
		require.NoError(t, err)
		require.Equal(t, 200, code)

		af, _, err = m.Login(&AuthForm{
			AccountID: id,
			Empno:     "100",
			Password:  "100000000",
		})
		require.NoError(t, err)

		//change password
		modMng.Password = "100000001"
		_, err = m.UpdateManager(af, modMng)
		require.NoError(t, err)
		_, code, err = m.RefreshToken(af)
		require.Equal(t, 401, code)
		require.EqualError(t, err, "用户登录信息已失效")

		//409
		modMng.Empno = "0002"
		code, err = m.UpdateManager(af, modMng)
		require.EqualError(t, err, "UNIQUE constraint failed: users.account_id, users.empno")
		require.Equal(t, 409, code)

		//400
		modMng.Empno = "_(不是有效的工号)"
		code, err = m.UpdateManager(af, modMng)
		require.True(t, ginx.IsValidationError(err))
		require.Equal(t, 400, code)

		//404
		modMng.ID = zeroUUID
		modMng.Empno = "0009"
		code, err = m.UpdateManager(af, modMng)
		require.EqualError(t, err, "record not found")
		require.Equal(t, 404, code)
	})
}

func TestGetAccount(t *testing.T) {
	id, _, err := m.GetAccountIdByMobile("18609837575")
	require.NoError(t, err)

	t.Run("GetAccountByID", func(t *testing.T) {
		ac, code, err := m.GetAccountByID(af, id)

		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Equal(t, "百度baidu", ac.Corp)
		require.Equal(t, "18609837575", ac.Mobile)
		require.Equal(t, "公司：百度baidu, 手机：18609837575", ac.Remarks)
		require.Equal(t, 10, ac.MaxStores)
		require.Equal(t, 100, ac.MaxUsersPerStore)

		//400
		_, code, err = m.GetAccountByID(&AuthInfo{
			Empno: "root",
			Role:  AdminRole,
		}, "NotUUID")
		require.EqualError(t, err, "invalid UUID: NotUUID")
		require.Equal(t, 400, code)

		//404
		_, code, err = m.GetAccountByID(&AuthInfo{
			Empno: "root",
			Role:  AdminRole,
		}, zeroUUID)
		require.Error(t, err)
		require.Equal(t, 404, code)
	})

	t.Run("GetManagerOfAccount", func(t *testing.T) {
		ac, _, err := m.GetAccountByID(af, id)
		require.NoError(t, err)

		manager, code, err := m.GetManagerOfAccount(af, ac)
		ac.Manager = manager
		require.Equal(t, 200, code)
		require.Equal(t, "百度baidu", ac.Corp)
		require.Equal(t, "18609837575", ac.Mobile)
		require.Equal(t, "公司：百度baidu, 手机：18609837575", ac.Remarks)
		require.Equal(t, 10, ac.MaxStores)
		require.Equal(t, 100, ac.MaxUsersPerStore)
		require.NotNil(t, ac.Manager)
		require.Equal(t, ac.ManagerID, ac.Manager.ID)
		require.Equal(t, ac.ID, ac.Manager.AccountID)
		require.Equal(t, "李彦宏", ac.Manager.RealName)
		require.Equal(t, "000", ac.Manager.Empno)
		require.Equal(t, ManagerRole, ac.Manager.Role)

		//400
		_, code, err = m.GetManagerOfAccount(af, &Account{ManagerID:gm.UUID("NotUUID")})
		require.EqualError(t, err, "invalid UUID: NotUUID")
		require.Equal(t, 500, code)

		//404
		_, code, err = m.GetManagerOfAccount(af, &Account{ManagerID:zeroUUID})
		require.Error(t, err)
		require.Equal(t, 404, code)
	})

	t.Run("GetStoresOfAccount", func(t *testing.T) {
		ac, _, err := m.GetAccountByID(af, id)
		require.NoError(t, err)

		stores, code, err := m.GetStoresOfAccount(af, ac, false)
		require.Equal(t, 200, code)
		require.Equal(t, "百度baidu", ac.Corp)
		require.Equal(t, "18609837575", ac.Mobile)
		require.Equal(t, "公司：百度baidu, 手机：18609837575", ac.Remarks)
		require.Equal(t, 10, ac.MaxStores)
		require.Equal(t, 100, ac.MaxUsersPerStore)

		require.Len(t, stores, 3)
		for i := 0; i < 3; i++ {
			require.Equal(t, ac.ID, stores[i].AccountID)
			require.Equal(t, fmt.Sprint("分店", i), stores[i].Name)
			require.Equal(t, fmt.Sprint("分店名：分店", i), stores[i].Remarks)
		}

		//400
		_, code, err = m.GetManagerOfAccount(af, &Account{ManagerID:gm.UUID("NotUUID")})
		require.EqualError(t, err, "invalid UUID: NotUUID")
		require.Equal(t, 500, code)

		//404
		_, code, err = m.GetManagerOfAccount(af, &Account{ManagerID:zeroUUID})
		require.Error(t, err)
		require.Equal(t, 404, code)
	})
}

func TestGetStore(t *testing.T) {
	//get account
	id, _, err := m.GetAccountIdByMobile("18609837575")
	require.NoError(t, err)
	ac, _, err := m.GetAccountByID(af, id)
	require.NoError(t, err)

	//get stores
	stores, _, err := m.GetStoresOfAccount(af, ac, false)
	require.NoError(t, err)
	require.Len(t, stores, 3)

	t.Run("GetStoreByID", func(t *testing.T) {
		st, code, err := m.GetStoreByID(af, stores[0].ID)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Equal(t, "分店0", st.Name)
		require.Equal(t, "分店名：分店0", st.Remarks)

		//400
		_, code, err = m.GetStoreByID(af, emptyUUID)
		require.EqualError(t, err, "invalid UUID: ")
		require.Equal(t, 400, code)

		//404
		_, code, err = m.GetStoreByID(af, zeroUUID)
		require.Error(t, err)
		require.Equal(t, 404, code)
	})

	t.Run("GetUsersOfStore", func(t *testing.T) {
		st := &Store{}
		st.ID = stores[0].ID

		users, code, err := m.GetUsersOfStore(af, st, false)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Len(t, users, 3)

		for i := 0; i < 3; i++ {
			require.Equal(t, users[i].Empno, fmt.Sprint("000", i+1))
			require.Equal(t, users[i].RealName, fmt.Sprint("张", i))
			require.Equal(t, users[i].Role, fmt.Sprint("级别", i))
			require.Equal(t, users[i].StoreID.String(), st.ID.String())
			require.Equal(t, users[i].AccountID, ac.ID)
		}

		//500
		st.ID = emptyUUID
		_, code, err = m.GetUsersOfStore(af, st, false)
		require.EqualError(t, err, "invalid UUID: ")
		require.Equal(t, 500, code)
	})
}

func TestGetUser(t *testing.T) {
	//get account
	id, _, err := m.GetAccountIdByMobile("18609837575")
	require.NoError(t, err)
	ac, _, err := m.GetAccountByID(af, id)
	require.NoError(t, err)

	//get stores
	stores,_, err := m.GetStoresOfAccount(af, ac, false)
	require.NoError(t, err)
	require.Len(t, stores, 3)

	//get users
	st := &Store{}
	st.ID = stores[0].ID
	users, _, err := m.GetUsersOfStore(af, st, false)
	require.NoError(t, err)
	require.Len(t, users, 3)

	t.Run("ByID", func(t *testing.T) {
		usr, code, err := m.GetUserByID(af, users[0].ID)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Equal(t, "0001", usr.Empno)
		require.Equal(t, "张0", usr.RealName)
		require.Equal(t, "级别0", usr.Role)
		require.Equal(t, st.ID.String(), usr.StoreID.String())
		require.Equal(t, ac.ID, usr.AccountID)

		//manager
		usr, code, err = m.GetUserByID(af, ac.ManagerID)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Equal(t, "000", usr.Empno)
		require.Equal(t, "李彦宏", usr.RealName)
		require.Equal(t, ManagerRole, usr.Role)
		require.Nil(t, usr.StoreID)
		require.Equal(t, ac.ID, usr.AccountID)

		//400
		_, code, err = m.GetUserByID(af, emptyUUID)
		require.EqualError(t, err, "invalid UUID: ")
		require.Equal(t, 400, code)

		//404
		_, code, err = m.GetUserByID(af, zeroUUID)
		require.Error(t, err)
		require.Equal(t, 404, code)
	})

	t.Run("GetStore", func(t *testing.T) {
		//manager
		usr := &User{}
		usr.ID = ac.ManagerID
		tmp, code, err := m.GetStoreOfUser(af, usr)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Nil(t, tmp)

		//employee
		usr = &User{}
		usr.StoreID = &st.ID
		st1, code, err := m.GetStoreOfUser(af, usr)
		require.NoError(t, err)
		require.Equal(t, "分店0", st1.Name)
		require.Equal(t, "分店名：分店0", st1.Remarks)
		require.Equal(t, st.ID, st1.ID)

		//500

		usr.StoreID = &emptyUUID
		_, code, err = m.GetStoreOfUser(af, usr)
		require.EqualError(t, err, "invalid UUID: ")
		require.Equal(t, 500, code)
	})

	t.Run("GetAccount", func(t *testing.T) {
		//manager
		usr := &User{}
		usr.AccountID = ac.ID
		ac1, code, err := m.GetAccountOfUser(af, usr)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Equal(t, ac1.ID, ac.ID)
		require.Equal(t, "百度baidu", ac.Corp)
		require.Equal(t, "18609837575", ac.Mobile)
		require.Equal(t, "公司：百度baidu, 手机：18609837575", ac.Remarks)
		require.Equal(t, 10, ac.MaxStores)
		require.Equal(t, 100, ac.MaxUsersPerStore)

		//50
		usr.AccountID = emptyUUID
		_, code, err = m.GetAccountOfUser(af, usr)
		require.EqualError(t, err, "invalid UUID: ")
		require.Equal(t, 500, code)
	})
}

func getStore(accountID gm.UUID, index int) (*Store, error) {
	//get account
	ac, _, err := m.GetAccountByID(af, accountID)
	if err != nil {
		return nil, err
	}

	//get stores
	stores,_, err := m.GetStoresOfAccount(af, ac, false)
	if err != nil {
		return nil, err
	}
	if len(stores) < index+1 {
		return nil, err
	}

	return stores[index], nil
}

func TestDeleteAndReuse(t *testing.T) {
	id, _, err := m.GetAccountIdByMobile("18609837575")
	require.NoError(t, err)
	defer m.ReuseAccount(af, id)
	t.Run("Account", func(t *testing.T) {
		code, err := m.DeleteAccount(af, id)
		require.NoError(t, err)
		require.Equal(t, 200, code)

		acs := make([]Account, 0)
		err = m.db.Find(&acs).Error
		require.NoError(t, err)
		require.Len(t, acs, 2)

		//check deleted at
		ac, _, err := m.GetAccountByID(af, id)
		require.NoError(t, err)
		require.NotNil(t, ac.DeletedAt)
		require.GreaterOrEqual(t, time.Now().Unix(), ac.DeletedAt.Unix())

		//reuse
		code, err = m.ReuseAccount(af, id)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		err = m.db.Find(&acs).Error
		require.NoError(t, err)
		require.Len(t, acs, 3)

		//check deleted at
		ac, _, err = m.GetAccountByID(af, id)
		require.NoError(t, err)
		require.Nil(t, ac.DeletedAt)
	})

	t.Run("Store", func(t *testing.T) {
		//get account
		ac, _, err := m.GetAccountByID(af, id)
		require.NoError(t, err)

		//get stores
		stores, _, err := m.GetStoresOfAccount(af, ac, false)
		require.NoError(t, err)
		require.Len(t, stores, 3)

		code, err := m.DeleteStore(af, stores[0].ID)
		require.NoError(t, err)
		require.Equal(t, 200, code)

		stores, _, err = m.GetStoresOfAccount(af, ac, false)
		require.NoError(t, err)
		require.Len(t, stores, 2)
		for i := 0; i < 2; i++ {
			require.Equal(t, ac.ID, stores[i].AccountID)
			require.Equal(t, fmt.Sprint("分店", i+1), stores[i].Name)
			require.Equal(t, fmt.Sprint("分店名：分店", i+1), stores[i].Remarks)
		}

		//unscoped
		stores, _, err = m.GetStoresOfAccount(af, ac, true)
		require.NoError(t, err)
		require.Len(t, stores, 3)
		for i := 0; i < 3; i++ {
			require.Equal(t, ac.ID, stores[i].AccountID)
			require.Equal(t, fmt.Sprint("分店", i), stores[i].Name)
			require.Equal(t, fmt.Sprint("分店名：分店", i), stores[i].Remarks)
		}
		require.NotNil(t, stores[0].DeletedAt)
		require.GreaterOrEqual(t, time.Now().Unix(), stores[0].DeletedAt.Unix())

		//reuse
		code, err = m.ReuseStore(af, stores[0].ID)
		require.NoError(t, err)
		require.Equal(t, 200, code)

		st, code, err := m.GetStoreByID(af, stores[0].ID)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Nil(t, st.DeletedAt)
		require.Equal(t, ac.ID, st.AccountID)
		require.Equal(t, "分店0", st.Name)
		require.Equal(t, "分店名：分店0", st.Remarks)
	})

	t.Run("User", func(t *testing.T) {
		st, err := getStore(id, 0)
		require.NoError(t, err)

		//get users
		users, _, err := m.GetUsersOfStore(af, st, false)
		require.NoError(t, err)
		require.Len(t, users, 3)

		code, err := m.DeleteUser(af, users[0].ID)
		require.NoError(t, err)
		require.Equal(t, 200, code)

		users, _, err = m.GetUsersOfStore(af, st, false)
		require.NoError(t, err)
		require.Len(t, users, 2)
		for i := 0; i < 2; i++ {
			require.Equal(t, users[i].Empno, fmt.Sprint("000", i+2))
			require.Equal(t, users[i].RealName, fmt.Sprint("张", i+1))
			require.Equal(t, users[i].Role, fmt.Sprint("级别", i+1))
			require.Equal(t, users[i].StoreID.String(), st.ID.String())
			require.Equal(t, users[i].AccountID, id)
		}

		//unscoped
		users,_, err = m.GetUsersOfStore(af, st, true)
		require.NoError(t, err)
		require.Len(t, users, 3)
		for i := 0; i < 2; i++ {
			require.Equal(t, users[i].Empno, fmt.Sprint("000", i+1))
			require.Equal(t, users[i].RealName, fmt.Sprint("张", i))
			require.Equal(t, users[i].Role, fmt.Sprint("级别", i))
			require.Equal(t, users[i].StoreID.String(), st.ID.String())
			require.Equal(t, users[i].AccountID, id)
		}
		require.NotNil(t, users[0].DeletedAt)
		require.GreaterOrEqual(t, time.Now().Unix(), users[0].DeletedAt.Unix())

		//reuse
		code, err = m.ReuseUser(af, users[0].ID)
		require.NoError(t, err)
		require.Equal(t, 200, code)

		usr, code, err := m.GetUserByID(af, users[0].ID)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Nil(t, st.DeletedAt)
		require.Equal(t, usr.Empno, "0001")
		require.Equal(t, usr.RealName, "张0")
		require.Equal(t, usr.Role, "级别0")
		require.Equal(t, usr.StoreID.String(), st.ID.String())
		require.Equal(t, usr.AccountID, id)
	})
}

func TestSearchAccounts(t *testing.T) {
	t.Run("first and count", func(t *testing.T) {
		cnt := 0
		acts, code, err := m.SearchAccounts(af, nil, 2, nil, &cnt)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Equal(t, cnt, 3)
		require.Len(t, acts, 2)
	})

	t.Run("after", func(t *testing.T) {
		id, _, err := m.GetAccountIdByMobile("15932323232")
		require.NoError(t, err)
		acts, code, err := m.SearchAccounts(af, nil, 10, &id, nil)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.Len(t, acts, 2)
	})

	t.Run("time between", func(t *testing.T) {
		id, _, err := m.GetAccountIdByMobile("18609837575")
		require.NoError(t, err)
		_, err = m.DeleteAccount(af, id)
		require.NoError(t, err)
		defer m.ReuseAccount(af, id)

		t1 := time.Now().Add(-time.Hour)
		t2 := time.Now().Add(time.Hour)
		acts, _, err := m.SearchAccounts(af, &AccountFilter{
			Mobile:    "18",
			Unscoped:  1,
			CreatedAt: []*time.Time{&t1, &t2},
			DeletedAt: []*time.Time{&t1, &t2},
		}, 100, nil, nil)
		require.NoError(t, err)
		require.Len(t, acts, 1)
		require.Equal(t, acts[0].ID, id)
	})

	t.Run("like", func(t *testing.T) {
		min := 11
		max := 11
		acts, _, err := m.SearchAccounts(af, &AccountFilter{
			MaxStores:        []*int{nil, &max},
			MaxUsersPerStore: []*int{&min, nil},
			Orders:           []string{"max_stores", "-max_users_per_store", "-mobile"},
		}, 100, nil, nil)
		require.NoError(t, err)
		require.Len(t, acts, 3)
		require.Equal(t, acts[2].Mobile, "15932323232")
		require.Equal(t, acts[1].Mobile, "15987653275")
		require.Equal(t, acts[0].Mobile, "18609837575")

		//update account and test again
		modAc := &ModAccount{
			ID:               acts[0].ID,
			Mobile:           acts[0].Mobile,
			Corp:             acts[0].Corp,
			MaxStores:        9,
			MaxUsersPerStore: 800,
			ExpiresAt:        acts[0].ExpiresAt,
			Remarks:          acts[0].Remarks,
		}
		//update accounts
		_, err = m.UpdateAccount(af, modAc)
		require.NoError(t, err)

		min = 10
		max = 100
		acts, _, err = m.SearchAccounts(af, &AccountFilter{
			MaxStores:        []*int{&min},
			MaxUsersPerStore: []*int{&max},
			Orders:           []string{"mobile"},
		}, 100, nil, nil)
		require.NoError(t, err)
		require.Len(t, acts, 2)
		require.Equal(t, acts[0].Mobile, "15932323232")
		require.Equal(t, acts[1].Mobile, "15987653275")

		//clear
		modAc.MaxStores = 10
		modAc.MaxUsersPerStore = 100
		_, err = m.UpdateAccount(af, modAc)
		require.NoError(t, err)
	})

	t.Run("like", func(t *testing.T) {
		acts, _, err := m.SearchAccounts(af, &AccountFilter{
			Unscoped: 1,
			Corp:     "t",
			Mobile:   "159",
		}, 100, nil, nil)
		require.NoError(t, err)
		require.Len(t, acts, 2)
		require.Equal(t, acts[0].Mobile, "15932323232")
		require.Equal(t, acts[1].Mobile, "15987653275")

		acts, _, err = m.SearchAccounts(af, &AccountFilter{
			Unscoped: 1,
			Remarks:  "t",
			Mobile:   "75",
		}, 100, nil, nil)
		require.NoError(t, err)
		require.Len(t, acts, 1)
		require.Equal(t, acts[0].Mobile, "15987653275")
	})

	t.Run("order id error", func(t *testing.T) {
		min := 11
		max := 11
		_, _, err := m.SearchAccounts(af, &AccountFilter{
			MaxStores:        []*int{nil, &max},
			MaxUsersPerStore: []*int{&min, nil},
			Orders:           []string{"id", "-id", "-mobile"},
		}, 100, nil, nil)
		require.True(t, ginx.IsValidationError(err))
	})
}
