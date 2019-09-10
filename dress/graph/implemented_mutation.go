package graph

import (
	"context"
	"github.com/zltgo/dress/model"
	gm "github.com/zltgo/webkit/graphorm"
	"net/http"
)

func (r *mutationResolver) NewAccount(ctx context.Context, input model.NewAccount) (*model.Account, error) {
	m, af, err := r.GetAuthInfo(ctx, "CreateAccount")
	if err != nil {
		return nil, err
	}

	ac, code, err := m.CreateAccount(af, &input)
	if err != nil {
		var message string
		if code == http.StatusConflict {
			message = "账户手机号已存在"
		}
		return nil, gm.NewError(code, err, message)
	}
	return ac, nil
}
func (r *mutationResolver) NewUser(ctx context.Context, input model.NewUser) (*model.User, error) {
	m, af, err := r.GetAuthInfo(ctx,"CreateUser")
	if err != nil {
		return nil, err
	}

	ac, code, err := m.CreateUser(af, &input)
	if err != nil {
		var message string
		switch code {
		case http.StatusConflict:
			message = "工号已存在"
		case http.StatusPreconditionFailed:
			message = "该分店员工数量已达上限"
		}
		return nil, gm.NewError(code, err, message)
	}
	return ac, nil
}
func (r *mutationResolver) NewStore(ctx context.Context, input model.NewStore) (*model.Store, error) {
	m, af, err := r.GetAuthInfo(ctx,"CreateStore")
	if err != nil {
		return nil, err
	}

	ac, code, err := m.CreateStore(af, &input)
	if err != nil {
		var message string
		switch code {
		case http.StatusConflict:
			message = "分店名称已存在"
		case http.StatusPreconditionFailed:
			message = "分店数量已达上限"
		}
		return nil, gm.NewError(code, err, message)
	}
	return ac, nil
}

func (r *mutationResolver) ModAccount(ctx context.Context, input model.ModAccount) (bool, error) {
	m, af, err := r.GetAuthInfo(ctx, "UpdateAccount")
	if err != nil {
		return false, err
	}

	code, err := m.UpdateAccount(af, &input)
	if err != nil {
		var message string
		if code == http.StatusConflict {
			message = "账户手机号已存在"
		}
		return false, gm.NewError(code, err, message)
	}

	return true, nil
}

func (r *mutationResolver) ModUser(ctx context.Context, input model.ModUser) (bool, error) {
	m, af, err := r.GetAuthInfo(ctx, "UpdateUser")
	if err != nil {
		return false, err
	}

	code, err := m.UpdateUser(af, &input)
	if err != nil {
		var message string
		if code == http.StatusConflict {
			message = "工号已存在"
		}
		return false, gm.NewError(code, err, message)
	}
	return true, nil
}

func (r *mutationResolver) ModStore(ctx context.Context, input model.ModStore) (bool, error) {
	m, af, err := r.GetAuthInfo(ctx, "UpdateStore")
	if err != nil {
		return false, err
	}

	code, err := m.UpdateStore(af, &input)
	if err != nil {
		var message string
		if code == http.StatusConflict {
			message = "分店名称已存在"
		}
		return false, gm.NewError(code, err, message)
	}
	return true, nil
}

func (r *mutationResolver) ModManager(ctx context.Context, input model.ModManager) (bool, error) {
	m, af, err := r.GetAuthInfo(ctx, "UpdateManager")
	if err != nil {
		return false, err
	}

	code, err := m.UpdateManager(af, &input)
	if err != nil {
		var message string
		if code == http.StatusConflict {
			message = "工号已存在"
		}
		return false, gm.NewError(code, err, message)
	}
	return true, nil
}

func (r *mutationResolver) DisableAccount(ctx context.Context, accountID gm.UUID) (bool, error) {
	m, af, err := r.GetAuthInfo(ctx, "DeleteAccount")
	if err != nil {
		return false, err
	}

	if code, err := m.DeleteAccount(af, accountID); err != nil {
		return false, gm.NewError(code, err)
	}
	return true, nil
}
func (r *mutationResolver) DisableUser(ctx context.Context, userID gm.UUID) (bool, error) {
	m, af, err := r.GetAuthInfo(ctx, "DeleteUser")
	if err != nil {
		return false, err
	}

	if code, err := m.DeleteUser(af, userID); err != nil {
		return false, gm.NewError(code, err)
	}
	return true, nil
}
func (r *mutationResolver) DisableStore(ctx context.Context, storeID gm.UUID) (bool, error) {
	m, af, err := r.GetAuthInfo(ctx, "DeleteStore")
	if err != nil {
		return false, err
	}

	if code, err := m.DeleteStore(af, storeID); err != nil {
		return false, gm.NewError(code, err)
	}
	return true, nil
}

func (r *mutationResolver) EnableAccount(ctx context.Context, accountID gm.UUID) (bool, error) {
	m, af, err := r.GetAuthInfo(ctx, "ReuseAccount")
	if err != nil {
		return false, err
	}

	if code, err := m.ReuseAccount(af, accountID); err != nil {
		return false, gm.NewError(code, err)
	}
	return true, nil
}
func (r *mutationResolver) EnableUser(ctx context.Context, userID gm.UUID) (bool, error) {
	m, af, err := r.GetAuthInfo(ctx, "ReuseUser")
	if err != nil {
		return false, err
	}

	if code, err := m.ReuseUser(af, userID); err != nil {
		return false, gm.NewError(code, err)
	}
	return true, nil
}
func (r *mutationResolver) EnableStore(ctx context.Context, storeID gm.UUID) (bool, error) {
	m, af, err := r.GetAuthInfo(ctx, "ReuseStore")
	if err != nil {
		return false, err
	}

	if code, err := m.ReuseStore(af, storeID); err != nil {
		return false, gm.NewError(code, err)
	}
	return true, nil
}
