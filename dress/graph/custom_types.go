package graph

import (
	"github.com/zltgo/dress/model"
	gm "github.com/zltgo/webkit/graphorm"
)

type AccountsConnection struct {
	TotalCount int             `json:"totalCount"`
	Accounts   []model.Account `json:"accounts"`
}

type AccountsEdge struct {
	Cursor gm.UUID        `json:"cursor"`
	Node   *model.Account `json:"node"`
}

// Edges returns PageInfo of AccountsConnection
func (ac *AccountsConnection) PageInfo() PageInfo {
	if size := len(ac.Accounts); size > 0 {
		return PageInfo{
			StartCursor: ac.Accounts[0].ID,
			EndCursor:   ac.Accounts[size-1].ID,
			HasNextPage: ac.TotalCount-size > 0,
		}
	}

	return PageInfo{
		StartCursor: gm.ZeroUUID(),
		EndCursor:   gm.ZeroUUID(),
		HasNextPage: ac.TotalCount > 0,
	}
}

// Edges returns edges of AccountsConnection
func (ac *AccountsConnection) Edges() []AccountsEdge {
	egs := make([]AccountsEdge, len(ac.Accounts))
	for i := 0; i < len(ac.Accounts); i++ {
		egs[i].Cursor = ac.Accounts[i].ID
		egs[i].Node = &ac.Accounts[i]
	}
	return egs
}
