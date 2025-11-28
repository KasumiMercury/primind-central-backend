package user

import (
	"context"
	"errors"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	SaveUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id ID) (*User, error)
}
