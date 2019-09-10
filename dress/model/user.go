package model

import (
	gm "github.com/zltgo/webkit/graphorm"
	"github.com/zltgo/webkit/jwt"
	"golang.org/x/exp/errors/fmt"
	"net/http"
	"time"
)

// Get AccountID by mobile number
// errors ：400,404,500
// scoped
func (m *Model) GetAccountIdByMobile(mobile string) (gm.UUID, int, error) {
	//validation
	if err := m.validator.Var(mobile, "mobile"); err != nil {
		return gm.ZeroUUID(), http.StatusBadRequest, err
	}

	rv := struct{ ID gm.UUID }{gm.ZeroUUID()}
	err := m.db.Model(&Account{}).Select("id").Where("mobile = ?", mobile).Scan(&rv).Error
	return rv.ID, gm.ErrorCode(err), err
}

// login and get token
// errors ：401
func (m *Model) Admin(name, password string) (*AuthInfo, int, error) {
	admin := m.opts.Admin
	if name != admin.Name || gm.NewMD5(name+password).String() != admin.PasswordHash {
		return nil, http.StatusUnauthorized, fmt.Errorf("name or password error")
	}

	return &AuthInfo{
		Empno:        admin.Name,
		PasswordHash: admin.PasswordHash,
		Role:         AdminRole,
		ExpiresAt:    jwt.TimeNow().Add(time.Duration(admin.MaxAge) * time.Second),
	}, http.StatusOK, nil
}

// login and get token
// errors ：400,401,402,404,500
// scoped
func (m *Model) Login(af *AuthForm) (*AuthInfo, int, error) {
	//validation
	if err := m.validator.ValidateStruct(af); err != nil {
		return nil, http.StatusBadRequest, err
	}

	user := User{}
	if err := m.db.Where("account_id = ? AND empno = ?", af.AccountID, af.Empno).First(&user).Error; err != nil {
		return nil, gm.ErrorCode(err), err
	}

	//验证密码是否正确，数据库存储的是password+salt的md5值
	if gm.NewMD5(af.Password+user.Salt) != user.PasswordHash {
		return nil, http.StatusUnauthorized, fmt.Errorf("name or password error")
	}

	//获取账号信息, 使用缓存
	var ac *Account
	if err := m.GetFromCacheOrDB(af.AccountID, &ac); err != nil {
		return nil, gm.ErrorCode(err), err
	}
	if ac.DeletedAt != nil {
		return nil, http.StatusNotFound, fmt.Errorf("account disabled")
	}

	//检查有效期
	if ac.ExpiresAt.Before(jwt.TimeNow()) {
		return nil, http.StatusPaymentRequired, fmt.Errorf("account expired: %v", ac.ExpiresAt)
	}

	return &AuthInfo{
		UserId:       user.ID,
		AccountID:    user.AccountID,
		Empno:        user.Empno,
		PasswordHash: user.PasswordHash.String(),
		Role:         user.Role,
		ExpiresAt:    ac.ExpiresAt,
	}, http.StatusOK, nil
}

// 刷新token
// errors: 404,500
// scoped
func (m *Model) RefreshToken(af *AuthInfo) (*AuthInfo, int, error) {
	//admin
	if af.Role == AdminRole {
		if af.Empno != m.opts.Admin.Name || af.PasswordHash != m.opts.Admin.PasswordHash {
			return nil, http.StatusUnauthorized, fmt.Errorf("管理员登录信息已失效")
		}
		if af.ExpiresAt.Before(jwt.TimeNow()) {
			return nil, http.StatusUnauthorized, fmt.Errorf("admin expired: %v", af.ExpiresAt)
		}
		return af, http.StatusOK, nil
	}

	//user
	var user *User
	if err := m.GetFromCacheOrDB(af.UserId, &user); err != nil {
		return nil, gm.ErrorCode(err), err
	}
	if user.DeletedAt != nil {
		return nil, http.StatusNotFound, fmt.Errorf("user disabled")
	}

	//检查密码是否已经更动，改动以后登录信息不再有效
	if af.PasswordHash != user.PasswordHash.String() {
		return nil, http.StatusUnauthorized, fmt.Errorf("用户登录信息已失效")
	}

	//获取account信息
	var ac *Account
	if err := m.GetFromCacheOrDB(user.AccountID, &ac); err != nil {
		return nil, gm.ErrorCode(err), err
	}
	if ac.DeletedAt != nil {
		return nil, http.StatusNotFound, fmt.Errorf("account disabled")
	}

	//检查有效期
	if ac.ExpiresAt.Before(jwt.TimeNow()) {
		return nil, http.StatusPaymentRequired, fmt.Errorf("account expired: %v", ac.ExpiresAt)
	}

	return &AuthInfo{
		UserId:       user.ID,
		AccountID:    user.AccountID,
		Empno:        user.Empno,
		PasswordHash: user.PasswordHash.String(),
		Role:         user.Role,
		ExpiresAt:    ac.ExpiresAt,
	}, http.StatusOK, nil
}

// Get manager of the account
// errors ：400,403,404,500
// unscoped
func (m *Model) GetAccountByID(af *AuthInfo, id gm.UUID) (*Account, int, error) {
	//ensure id is valid
	if !id.IsValid() {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", id.String())
	}

	var ac *Account
	if err := m.GetFromCacheOrDB(id, &ac); err != nil {
		return nil, gm.ErrorCode(err), err
	}
	return ac, http.StatusOK, nil
}

// Get manager of the account
// errors ：403,404,500
// unscoped
func (m *Model) GetManagerOfAccount(af *AuthInfo, ac *Account) (*User, int, error) {
	var manager *User
	if err := m.GetFromCacheOrDB(ac.ManagerID, &manager); err != nil {
		return nil, gm.ErrorCode(err), err
	}
	return manager, http.StatusOK, nil
}

// Get stores of the account
// errors ：403,500
func (m *Model) GetStoresOfAccount(af *AuthInfo, ac *Account, unscoped bool) ([]*Store, int, error) {
	//ensure the list is not nil
	stores := make([]*Store, 0)
	db := m.db.Scopes(gm.Unscoped(unscoped)).Order("id")
	if err := db.Model(&Store{}).Where("account_id = ?", ac.ID).Find(&stores).Error; gm.IsInternalServerError(err) {
		return nil, http.StatusInternalServerError, err
	}
	return stores, http.StatusOK, nil
}

// Get user by id
// errors ：400,403,404,500
// unscoped
func (m *Model) GetUserByID(af *AuthInfo, id gm.UUID) (*User, int, error) {
	//ensure id is valid
	if !id.IsValid() {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", id.String())
	}

	var usr *User
	if err := m.GetFromCacheOrDB(id, &usr); err != nil {
		return nil, gm.ErrorCode(err), err
	}
	return usr, http.StatusOK, nil
}

// Get store of the user belongs to.
// errors ：403,500
// unscoped
func (m *Model) GetStoreOfUser(af *AuthInfo, usr *User) (*Store, int, error) {
	if usr.StoreID == nil {
		return nil, http.StatusOK, nil
	}

	var st *Store
	if err := m.GetFromCacheOrDB(*usr.StoreID, &st); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return st, http.StatusOK, nil
}

// Get account of the user belongs to.
// errors ：403,404,500
// unscoped
func (m *Model) GetAccountOfUser(af *AuthInfo, usr *User) (*Account, int, error) {
	var ac *Account
	if err := m.GetFromCacheOrDB(usr.AccountID, &ac); err != nil {
		return nil, gm.ErrorCode(err), err
	}
	return ac, http.StatusOK, nil
}

// Get store by id
// errors ：400,404,500
// unscoped
func (m *Model) GetStoreByID(af *AuthInfo, id gm.UUID) (*Store, int, error) {
	//ensure id is valid
	if !id.IsValid() {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", id.String())
	}

	var st *Store
	if err := m.GetFromCacheOrDB(id, &st); err != nil {
		return nil, gm.ErrorCode(err), err
	}
	return st, http.StatusOK, nil
}

// Get all users of the store.
// errors ：403,500
func (m *Model) GetUsersOfStore(af *AuthInfo, store *Store, unscoped bool) ([]*User, int, error) {
	//ensure the list is not nil
	users := make([]*User, 0)
	db := m.db.Scopes(gm.Unscoped(unscoped)).Order("id")
	if err := db.Model(&User{}).Where("store_id = ?", store.ID).Find(&users).Error; gm.IsInternalServerError(err) {
		return nil, http.StatusInternalServerError, err
	}
	return users, http.StatusOK, nil
}

// Search accounts by filter, if totalCnt isn't nil, total count will return
// errors ：400,403,500
func (m *Model) SearchAccounts(af *AuthInfo, filter *AccountFilter, first int, after *gm.UUID, totalCnt *int) ([]Account, int, error) {
	//validation
	if first <= 0 || first > m.opts.QueryLimit {
		return nil, http.StatusBadRequest, fmt.Errorf("查询数目超过最多%d条限制: %d", m.opts.QueryLimit, first)
	}
	if after != nil && !after.IsValid() {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", after.String())
	}

	//conditions,排序以先设置的为准,id必须保证顺序排序
	myDb := m.db
	if filter != nil {
		//validation
		if err := m.validator.ValidateStruct(filter); err != nil {
			return nil, http.StatusBadRequest, err
		}
		//range
		myDb = myDb.Scopes(gm.Between("created_at", filter.CreatedAt),
			gm.Between("updated_at", filter.UpdatedAt),
			gm.Between("deleted_at", filter.DeletedAt),
			gm.Between("expires_at", filter.ExpiresAt),
			gm.Between("max_stores", filter.MaxStores),
			gm.Between("max_users_per_store", filter.MaxUsersPerStore),
			gm.Like("corp", filter.Corp),
			gm.Like("mobile", filter.Mobile),
			gm.Like("remarks", filter.Remarks),
			gm.Unscoped(filter.Unscoped),
			gm.OrderBy(filter.Orders),
		)
	}
	// must place after filter
	myDb = myDb.Order("id")
	//afterId
	if after != nil {
		myDb = myDb.Where("id > ?", *after)
	}

	//count
	if totalCnt != nil {
		*totalCnt = 0 //init
		if err := myDb.Model(&Account{}).Count(totalCnt).Error; err != nil {
			return nil, gm.ErrorCode(err), err
		}
	}

	//query
	rv := make([]Account, 0)
	if err := myDb.Limit(first).Find(&rv).Error; gm.IsInternalServerError(err) {
		return nil, http.StatusInternalServerError, err
	}
	return rv, http.StatusOK, nil
}

// CreateAccount create a account record including it's manager
// errors ：400,403,409,500
func (m *Model) CreateAccount(af *AuthInfo, input *NewAccount) (*Account, int, error) {
	//validation
	if err := m.validator.ValidateStruct(input); err != nil {
		return nil, http.StatusBadRequest, err
	}

	accountId := gm.NewUUID()
	salt := jwt.RandString(16)
	ac := Account{
		Model:            gm.Model{ID: accountId},
		Corp:             input.Corp,
		Mobile:           input.Mobile,
		MaxStores:        input.MaxStores,
		MaxUsersPerStore: input.MaxUsersPerStore,
		Remarks:          input.Remarks,
		ExpiresAt:        input.ExpiresAt,
		Manager: &User{
			AccountID:    accountId,
			Empno:        input.Empno,
			PasswordHash: gm.NewMD5(input.Password + salt),
			Salt:         salt,
			RealName:     input.RealName,
			Role:         ManagerRole,
		},
	}

	if err := m.db.Create(&ac).Error; err != nil {
		return nil, gm.ErrorCode(err), err
	}
	return &ac, http.StatusOK, nil
}

// CreateStore create a store record
// errors ：400,403,409,412,500
func (m *Model) CreateStore(af *AuthInfo, input *NewStore) (*Store, int, error) {
	//validation
	if err := m.validator.ValidateStruct(input); err != nil {
		return nil, http.StatusBadRequest, err
	}

	//check maxStore
	if code, err := m.checkMaxStores(input.AccountID); err != nil {
		return nil, code, err
	}

	st := Store{
		AccountID: input.AccountID,
		Name:      input.Name,
		Remarks:   input.Remarks,
	}

	if err := m.db.Create(&st).Error; err != nil {
		return nil, gm.ErrorCode(err), err
	}
	return &st, http.StatusOK, nil
}

//check checkMaxStores
func (m *Model) checkMaxStores(accountID gm.UUID) (int, error) {
	cnt := 0
	if err := m.db.Model(&Store{}).Where("account_id = ?", accountID).Count(&cnt).Error; err != nil {
		return gm.ErrorCode(err), err
	}
	rv := struct{ MaxStores int }{}
	if err := m.db.Model(&Account{}).Select("max_stores").Where("id = ?", accountID).Scan(&rv).Error; err != nil {
		return gm.ErrorCode(err), err
	}
	if cnt >= rv.MaxStores {
		return http.StatusPreconditionFailed, fmt.Errorf("账户(ID:%v)创建的分店数量已达上限：%d", accountID, rv.MaxStores)
	}
	return http.StatusOK, nil
}

//check maxUsersPerStore
func (m *Model) checkMaxUsersPerStore(accountID, storeID gm.UUID) (int, error) {
	cnt := 0
	if err := m.db.Model(&User{}).Where("store_id = ?", storeID).Count(&cnt).Error; err != nil {
		return gm.ErrorCode(err), err
	}

	rv := struct{ MaxUsersPerStore int }{}
	if err := m.db.Model(&Account{}).Select("max_users_per_store").Where("id = ?", accountID).Scan(&rv).Error; err != nil {
		return gm.ErrorCode(err), err
	}
	if cnt >= rv.MaxUsersPerStore {
		return http.StatusPreconditionFailed, fmt.Errorf("分店(ID:%v)内的员工数量已达上限：%d", storeID, rv.MaxUsersPerStore)
	}
	return http.StatusOK, nil
}

// CreateAccount create a user record
// errors ：400,403,409,412,500
func (m *Model) CreateUser(af *AuthInfo, input *NewUser) (*User, int, error) {
	//validation
	if err := m.validator.ValidateStruct(input); err != nil {
		return nil, http.StatusBadRequest, err
	}
	if input.Role == AdminRole || input.Role == ManagerRole {
		return nil, http.StatusBadRequest, fmt.Errorf("普通用户不能设置为管理员权限")
	}

	//check maxUsersPerStore
	if code, err := m.checkMaxUsersPerStore(input.AccountID, input.StoreID); err != nil {
		return nil, code, err
	}

	salt := jwt.RandString(16)
	usr := User{
		AccountID:    input.AccountID,
		StoreID:      &input.StoreID,
		Empno:        input.Empno,
		PasswordHash: gm.NewMD5(input.Password + salt),
		Salt:         salt,
		RealName:     input.RealName,
		Role:         input.Role,
	}

	if err := m.db.Create(&usr).Error; err != nil {
		return nil, gm.ErrorCode(err), err
	}
	return &usr, http.StatusOK, nil
}

// errors ：400,403,404,409,500
func (m *Model) UpdateAccount(af *AuthInfo, input *ModAccount) (int, error) {
	//validation
	if err := m.validator.ValidateStruct(input); err != nil {
		return http.StatusBadRequest, err
	}

	rv := m.db.Model(&Account{}).Update(input)
	if rv.Error != nil {
		return gm.ErrorCode(rv.Error), rv.Error
	}
	if rv.RowsAffected == 0 {
		return http.StatusNotFound, fmt.Errorf("record not found")
	}

	//delete old value from cache
	m.cacheDB.Remove(input.ID)

	return http.StatusOK, nil
}

// errors ：400,403,404,409,500
func (m *Model) UpdateStore(af *AuthInfo, input *ModStore) (int, error) {
	//validation
	if err := m.validator.ValidateStruct(input); err != nil {
		return http.StatusBadRequest, err
	}

	rv := m.db.Model(&Store{}).Update(input)
	if rv.Error != nil {
		return gm.ErrorCode(rv.Error), rv.Error
	}
	if rv.RowsAffected == 0 {
		return http.StatusNotFound, fmt.Errorf("record not found")
	}

	//delete old value from cache
	m.cacheDB.Remove(input.ID)
	return http.StatusOK, nil
}

// errors ：400,403,404,409,500
func (m *Model) UpdateUser(af *AuthInfo, input *ModUser) (int, error) {
	//validation
	if err := m.validator.ValidateStruct(input); err != nil {
		return http.StatusBadRequest, err
	}
	if input.Role == AdminRole || input.Role == ManagerRole {
		return http.StatusBadRequest, fmt.Errorf("普通用户不能设置为管理员权限")
	}

	//generate password hash
	if input.Password != "" {
		//read salt
		rv := struct{ Salt string }{}
		if err := m.db.Model(&User{}).Select("salt").Where("id = ?", input.ID).Scan(&rv).Error; err != nil {
			return gm.ErrorCode(err), err
		}
		input.PasswordHash = gm.NewMD5(input.Password + rv.Salt)
	}

	db := m.db.Model(&User{}).Update(input)
	if db.Error != nil {
		return gm.ErrorCode(db.Error), db.Error
	}
	if db.RowsAffected == 0 {
		return http.StatusNotFound, fmt.Errorf("record not found")
	}

	//delete old value from cache
	m.cacheDB.Remove(input.ID)
	return http.StatusOK, nil
}

// errors ：400,403,404,409,500
func (m *Model) UpdateManager(af *AuthInfo, input *ModManager) (int, error) {
	//validation
	if err := m.validator.ValidateStruct(input); err != nil {
		return http.StatusBadRequest, err
	}

	//generate password hash
	if input.Password != "" {
		//read salt
		rv := struct{ Salt string }{}
		if err := m.db.Model(&User{}).Select("salt").Where("id = ?", input.ID).Scan(&rv).Error; err != nil {
			return gm.ErrorCode(err), err
		}
		input.PasswordHash = gm.NewMD5(input.Password + rv.Salt)
	}

	db := m.db.Model(&User{}).Update(input)
	if db.Error != nil {
		return gm.ErrorCode(db.Error), db.Error
	}
	if db.RowsAffected == 0 {
		return http.StatusNotFound, fmt.Errorf("record not found")
	}

	//delete old value from cache
	m.cacheDB.Remove(input.ID)
	return http.StatusOK, nil
}

// RemoveAccount removes Account and all users and stores.
// if id is not exist, no errors return.
// errors ：400,403,500
//func (m *Model) RemoveAccount(af *AuthInfo, id gm.UUID) (code int, err error) {
//	//check permission
//	if code, err = m.CheckPermission(af, "RemoveAccount"); err != nil {
//		return code, err
//	}
//	//ensure id is valid
//	if !id.IsValid() {
//		return http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", id.String())
//	}
//
//	tx := m.db.Begin()
//	if tx.Error != nil {
//		return http.StatusInternalServerError, err
//	}
//
//	//in case of panic, ensure rollback is called
//	defer func() {
//		if r := recover(); r != nil {
//			tx.Rollback()
//			code = http.StatusInternalServerError
//			err = fmt.Errorf("panics: %v", r)
//		}
//	}()
//
//	//activate account by id
//	ac := Account{}
//	ac.ID = id
//	if err = tx.Delete(&ac).Error; err != nil {
//		tx.Rollback()
//		return gm.ErrorCode(err), err
//	}
//
//	//delete users of the account
//	if err = tx.Where("account_id = ?", id).Delete(&User{}).Error; err != nil {
//		tx.Rollback()
//		return gm.ErrorCode(err), err
//	}
//
//	//delete stores of the account
//	if err = tx.Where("account_id = ?", id).Delete(&Store{}).Error; err != nil {
//		tx.Rollback()
//		return gm.ErrorCode(err), err
//	}
//
//	//commit
//	err = tx.Commit().Error
//	return gm.ErrorCode(err), err
//}

// DeleteAccount removes Account.
// if id is not exist, no errors return.
// errors ：400,403,500
func (m *Model) DeleteAccount(af *AuthInfo, id gm.UUID) (code int, err error) {
	//ensure id is valid
	if !id.IsValid() {
		return http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", id.String())
	}

	//delete account by id
	if err := m.db.Where("id = ?", id).Delete(&Account{}).Error; err != nil {
		return gm.ErrorCode(err), err
	}

	//delete old value from cache
	m.cacheDB.Remove(id)
	return http.StatusOK, nil
}

// DeleteAccount removes a user record.
// Can not remove manager.
// errors：400,403,500
func (m *Model) DeleteUser(af *AuthInfo, id gm.UUID) (code int, err error) {
	//ensure id is valid
	if !id.IsValid() {
		return http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", id.String())
	}

	//delete user by id
	if err := m.db.Where("id = ? and role <> ?", id, ManagerRole).Delete(&User{}).Error; err != nil {
		return gm.ErrorCode(err), err
	}

	//delete old value from cache
	m.cacheDB.Remove(id)
	return http.StatusOK, nil
}

// DeleteAccount removes a Store record.
// errors：400,403,500
func (m *Model) DeleteStore(af *AuthInfo, id gm.UUID) (code int, err error) {
	//ensure id is valid
	if !id.IsValid() {
		return http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", id.String())
	}

	//delete store by id
	if err := m.db.Where("id = ?", id).Delete(&Store{}).Error; err != nil {
		return gm.ErrorCode(err), err
	}

	//delete old value from cache
	m.cacheDB.Remove(id)
	return http.StatusOK, nil
}

// ReuseAccount reuse Account.
// errors ：400,403,404,500
func (m *Model) ReuseAccount(af *AuthInfo, id gm.UUID) (code int, err error) {
	//ensure id is valid
	if !id.IsValid() {
		return http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", id.String())
	}

	//activate account by id
	ac := &Account{}
	ac.ID = id
	db := m.db.Unscoped().Model(ac).Update("deleted_at", nil)
	if db.Error != nil {
		return gm.ErrorCode(db.Error), db.Error
	}
	if db.RowsAffected == 0 {
		return http.StatusNotFound, fmt.Errorf("record not found")
	}

	//delete old value from cache
	m.cacheDB.Remove(id)
	return http.StatusOK, nil
}

// ReuseAccount reuse a user record.
// Can not remove manager.
// errors：400,403,404,500
func (m *Model) ReuseUser(af *AuthInfo, id gm.UUID) (code int, err error) {
	//ensure id is valid
	if !id.IsValid() {
		return http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", id.String())
	}

	var usr *User
	if usr, code, err = m.GetUserByID(af, id); err != nil {
		return code, err
	}
	//manager
	if usr.Role == ManagerRole {
		err = m.db.Unscoped().Model(usr).Update("deleted_at", nil).Error
		return gm.ErrorCode(err), err
	}

	//check maxUsersPerStore
	if code, err = m.checkMaxUsersPerStore(usr.AccountID, *usr.StoreID); err != nil {
		return code, err
	}

	//activate account by id
	if err := m.db.Unscoped().Model(usr).Update("deleted_at", nil).Error; err != nil {
		return gm.ErrorCode(err), err
	}

	//delete old value from cache
	m.cacheDB.Remove(id)
	return http.StatusOK, nil
}

// ReuseAccount reuse a Store record.
// errors：400,403,500
func (m *Model) ReuseStore(af *AuthInfo, id gm.UUID) (code int, err error) {
	//ensure id is valid
	if !id.IsValid() {
		return http.StatusBadRequest, fmt.Errorf("invalid UUID: %s", id.String())
	}

	var st *Store
	if st, code, err = m.GetStoreByID(af, id); err != nil {
		return code, err
	}
	//check maxStores
	if code, err = m.checkMaxStores(st.AccountID); err != nil {
		return code, err
	}

	//activate account by id
	if err := m.db.Unscoped().Model(st).Update("deleted_at", nil).Error; err != nil {
		return gm.ErrorCode(err), err
	}

	//delete old value from cache
	m.cacheDB.Remove(id)
	return http.StatusOK, nil
}
