package user

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewIDSuccess(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "generates valid UUIDv7"},
		{name: "generates non-empty ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewID()
			if err != nil {
				t.Fatalf("NewID() unexpected error: %v", err)
			}

			if id.String() == "" {
				t.Error("NewID() returned empty ID")
			}

			parsedUUID := uuid.UUID(id)
			if parsedUUID.Version() != 7 {
				t.Errorf("NewID() returned UUIDv%d, want v7", parsedUUID.Version())
			}
		})
	}
}

func TestNewIDFromStringSuccess(t *testing.T) {
	validID, err := NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	validIDStr := validID.String()

	tests := []struct {
		name  string
		input string
	}{
		{name: "valid UUIDv7 string", input: validIDStr},
		{name: "round-trip ID preservation", input: validIDStr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewIDFromString(tt.input)
			if err != nil {
				t.Fatalf("NewIDFromString(%q) unexpected error: %v", tt.input, err)
			}

			if id.String() != tt.input {
				t.Errorf("NewIDFromString(%q) = %q, want %q", tt.input, id.String(), tt.input)
			}

			parsedUUID := uuid.UUID(id)
			if parsedUUID.Version() != 7 {
				t.Errorf("NewIDFromString(%q) returned UUIDv%d, want v7", tt.input, parsedUUID.Version())
			}
		})
	}

	t.Run("round-trip consistency: NewID -> String -> NewIDFromString", func(t *testing.T) {
		originalID, err := NewID()
		if err != nil {
			t.Fatalf("NewID() error: %v", err)
		}

		idStr := originalID.String()
		parsedID, err := NewIDFromString(idStr)
		if err != nil {
			t.Fatalf("NewIDFromString(%q) error: %v", idStr, err)
		}

		if parsedID.String() != originalID.String() {
			t.Errorf("round-trip failed: got %q, want %q", parsedID.String(), originalID.String())
		}
	})
}

func TestNewIDFromStringErrors(t *testing.T) {
	uuidv4 := uuid.New()

	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name:        "empty string",
			input:       "",
			expectedErr: ErrIDInvalidFormat,
		},
		{
			name:        "invalid UUID format",
			input:       "not-a-uuid",
			expectedErr: ErrIDInvalidFormat,
		},
		{
			name:        "non-UUID string",
			input:       "12345678-abcd",
			expectedErr: ErrIDInvalidFormat,
		},
		{
			name:        "UUIDv4 instead of v7",
			input:       uuidv4.String(),
			expectedErr: ErrIDInvalidV7,
		},
		{
			name:        "malformed UUID with correct length",
			input:       "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			expectedErr: ErrIDInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewIDFromString(tt.input)

			if err == nil {
				t.Fatalf("NewIDFromString(%q) expected error, got ID: %s", tt.input, id.String())
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("NewIDFromString(%q) error = %v, want %v", tt.input, err, tt.expectedErr)
			}
		})
	}
}

func TestIDString(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "converts ID to string"},
		{name: "string is valid UUID format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewID()
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			idStr := id.String()
			if idStr == "" {
				t.Error("ID.String() returned empty string")
			}

			_, err = uuid.Parse(idStr)
			if err != nil {
				t.Errorf("ID.String() = %q is not valid UUID format: %v", idStr, err)
			}

			if !strings.Contains(idStr, "-") {
				t.Errorf("ID.String() = %q does not appear to be UUID format (missing dashes)", idStr)
			}
		})
	}
}

func TestNewUserSuccess(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates user with ID and color"},
		{name: "ID accessor returns correct value"},
		{name: "Color accessor returns correct value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewID()
			if err != nil {
				t.Fatalf("setup: NewID() error: %v", err)
			}

			color, err := NewColor("#FF5733")
			if err != nil {
				t.Fatalf("setup: NewColor() error: %v", err)
			}

			user := NewUser(id, color)

			if user == nil {
				t.Fatal("NewUser() returned nil")
			}

			if user.ID().String() != id.String() {
				t.Errorf("user.ID() = %q, want %q", user.ID().String(), id.String())
			}

			if user.Color().String() != color.String() {
				t.Errorf("user.Color() = %q, want %q", user.Color().String(), color.String())
			}
		})
	}
}

func TestCreateUserWithRandomColorSuccess(t *testing.T) {
	tests := []struct {
		name            string
		iterationCount  int
		checkUniqueness bool
	}{
		{name: "creates user with valid ID", iterationCount: 1, checkUniqueness: false},
		{name: "creates user with valid color", iterationCount: 1, checkUniqueness: false},
		{name: "generates 10 users with unique IDs", iterationCount: 10, checkUniqueness: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := make([]*User, 0, tt.iterationCount)
			ids := make(map[string]bool)

			for i := 0; i < tt.iterationCount; i++ {
				user, err := CreateUserWithRandomColor()
				if err != nil {
					t.Fatalf("CreateUserWithRandomColor() error on iteration %d: %v", i, err)
				}

				if user == nil {
					t.Fatal("CreateUserWithRandomColor() returned nil")
				}

				idStr := user.ID().String()
				if idStr == "" {
					t.Error("CreateUserWithRandomColor() returned user with empty ID")
				}

				parsedUUID := uuid.UUID(user.ID())
				if parsedUUID.Version() != 7 {
					t.Errorf("CreateUserWithRandomColor() returned user with UUIDv%d, want v7", parsedUUID.Version())
				}

				colorStr := user.Color().String()
				if colorStr == "" {
					t.Error("CreateUserWithRandomColor() returned user with empty color")
				}

				if !strings.HasPrefix(colorStr, "#") {
					t.Errorf("CreateUserWithRandomColor() returned user with invalid color format: %q (missing #)", colorStr)
				}

				if len(colorStr) != 7 {
					t.Errorf("CreateUserWithRandomColor() returned user with invalid color length: %q (want 7 chars)", colorStr)
				}

				users = append(users, user)

				if tt.checkUniqueness {
					if ids[idStr] {
						t.Errorf("CreateUserWithRandomColor() generated duplicate ID: %s", idStr)
					}
					ids[idStr] = true
				}
			}

			if tt.checkUniqueness && len(ids) != tt.iterationCount {
				t.Errorf("CreateUserWithRandomColor() generated %d unique IDs, want %d", len(ids), tt.iterationCount)
			}
		})
	}
}
