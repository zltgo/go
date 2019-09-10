package main

import (
	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
	"github.com/zltgo/webkit/graphorm/example/graph"
	"net/http/httptest"
	"testing"
)

type RawUser struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt, omitempty"`
	DeletedAt int64  `json:"deletedAt, omitempty"`
	Name      string `json:"name"`
}
type RawNewUser struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"createdAt"`
	DeletedAt int64  `json:"deletedAt, omitempty"`
	Name      string `json:"name"`
}

func TestScalars(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}})))
	c := client.New(srv.URL)
	t.Run("query", func(t *testing.T) {
		var resp struct {
			User RawUser
		}
		err := c.Post(`query GetUserByID{
				user(userId:"616161616161616161616161") {
					id
					createdAt
					updatedAt
					deletedAt
					name
				}
			}`, &resp)

		require.Equal(t, nil, err)
		require.Equal(t, "616161616161616161616161", resp.User.ID)
		require.Equal(t, int64(1000), resp.User.CreatedAt)
		require.Equal(t, int64(1000), resp.User.UpdatedAt)
		require.Equal(t, int64(0), resp.User.DeletedAt)
		require.Equal(t, "aaaaaaaaaaaa", resp.User.Name)
	})

	t.Run("mutation", func(t *testing.T) {
		input := RawNewUser{
			ID:        "626262626262626262626262",
			Name:      "bbbbbbbbbbbb",
			CreatedAt: 1001,
			DeletedAt: 1002,
		}

		var resp struct {
			NewUser RawUser
		}

		err := c.Post(`mutation CreateUser($u: NewUser){
					newUser(input:$u){
						id
						createdAt
						updatedAt
						deletedAt
						name	
					}
				}`,
			&resp,
			client.Var("u", input))

		require.Equal(t, nil, err)
		require.Equal(t, "626262626262626262626262", resp.NewUser.ID)
		require.Equal(t, int64(1001), resp.NewUser.CreatedAt)
		require.Equal(t, int64(1000), resp.NewUser.UpdatedAt)
		require.Equal(t, int64(1002), resp.NewUser.DeletedAt)
		require.Equal(t, "bbbbbbbbbbbb", resp.NewUser.Name)
	})
}
