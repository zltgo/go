// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package graph

import (
	"time"

	"github.com/zltgo/webkit/graphorm"
)

type NewUser struct {
	ID        graphorm.UUID `json:"id"`
	Name      string        `json:"name"`
	CreatedAt time.Time     `json:"createdAt"`
	DeletedAt *time.Time    `json:"deletedAt"`
}

type User struct {
	ID        graphorm.UUID `json:"id"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
	DeletedAt *time.Time    `json:"deletedAt"`
	Name      string        `json:"name"`
}
