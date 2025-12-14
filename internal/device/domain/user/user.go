package user

import (
	"fmt"

	"github.com/google/uuid"
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
