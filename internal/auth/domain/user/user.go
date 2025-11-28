package user

import "github.com/google/uuid"

type ID uuid.UUID

func NewID() ID {
	return ID(uuid.New())
}

func NewIDFromString(idStr string) (ID, error) {
	uuidVal, err := uuid.Parse(idStr)
	if err != nil {
		return ID{}, err
	}
	return ID(uuidVal), nil
}

type User struct {
	id ID
}

func NewUser(id ID) *User {
	return &User{
		id: id,
	}
}

func CreateUser() *User {
	return NewUser(NewID())
}

func (u *User) ID() ID {
	return u.id
}
