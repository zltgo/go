package graph

import (
	"context"
	"github.com/pkg/errors"
	"time"

	"github.com/zltgo/webkit/graphorm"
)

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type Resolver struct{}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) NewUser(ctx context.Context, input *NewUser) (*User, error) {
	if input.ID.IsValid() {
		return &User{
			ID:        input.ID,
			CreatedAt: input.CreatedAt,
			UpdatedAt: time.Unix(1000, 1000),
			DeletedAt: input.DeletedAt,
			Name:      input.Name,
		}, nil
	}
	return nil, errors.New("invalid uuid: " + input.ID.String())
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) User(ctx context.Context, userID graphorm.UUID) (*User, error) {
	v, err := userID.Value()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        userID,
		CreatedAt: time.Unix(1000, 1000),
		UpdatedAt: time.Unix(1000, 1000),
		DeletedAt: nil,
		Name:      string(v.([]byte)),
	}, nil
}
