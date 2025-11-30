package user

import "context"

type UserRepository interface {
	SaveUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id ID) (*User, error)
}
