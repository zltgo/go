// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package graph

import (
	"github.com/zltgo/webkit/graphorm"
)

type PageInfo struct {
	StartCursor graphorm.UUID `json:"startCursor"`
	EndCursor   graphorm.UUID `json:"endCursor"`
	HasNextPage bool          `json:"hasNextPage"`
}
