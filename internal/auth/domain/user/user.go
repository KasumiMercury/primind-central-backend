package user

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrIDGeneration    = errors.New("failed to generate user ID")
	ErrIDInvalidFormat = errors.New("user ID must be a valid UUID")
	ErrIDInvalidV7     = errors.New("user ID must be a UUIDv7")
)

type ID uuid.UUID

func NewID() (ID, error) {
	v7, err := uuid.NewV7()
	if err != nil {
		return ID{}, fmt.Errorf("%w: %v", ErrIDGeneration, err)
	}

	return ID(v7), nil
}

func NewIDFromString(idStr string) (ID, error) {
	uuidVal, err := uuid.Parse(idStr)
	if err != nil {
		return ID{}, fmt.Errorf("%w: %v", ErrIDInvalidFormat, err)
	}

	if uuidVal.Version() != 7 {
		return ID{}, ErrIDInvalidV7
	}

	return ID(uuidVal), nil
}

func (id ID) String() string {
	return uuid.UUID(id).String()
}

type User struct {
	id ID
}

func NewUser(id ID) *User {
	return &User{
		id: id,
	}
}

func CreateUser() (*User, error) {
	id, err := NewID()
	if err != nil {
		return nil, err
	}

	return NewUser(id), nil
}

func (u *User) ID() ID {
	return u.id
}
