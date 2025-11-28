package user

import "testing"

func TestNewColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid with hash uppercase",
			input: "#A1B2C3",
			want:  "#A1B2C3",
		},
		{
			name:  "valid lowercase with hash",
			input: "#a1b2c3",
			want:  "#A1B2C3",
		},
		{
			name:  "valid without hash",
			input: "00ff7a",
			want:  "#00FF7A",
		},
		{
			name:  "valid with spaces",
			input: "  #123abc ",
			want:  "#123ABC",
		},
		{
			name:    "invalid length",
			input:   "#FFF",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			input:   "#GGHHII",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewColor(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.String() != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestColorValidate(t *testing.T) {
	t.Parallel()

	valid := MustColor("#aabbcc")
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid color, got error: %v", err)
	}

	invalid := Color{hex: "#123"}
	if err := invalid.Validate(); err == nil {
		t.Fatalf("expected validation error for malformed color")
	}
}

func TestRandomPaletteColor(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{})

	for i := 0; i < 10; i++ {
		color, err := RandomPaletteColor()
		if err != nil {
			t.Fatalf("expected color, got error: %v", err)
		}

		if err := color.Validate(); err != nil {
			t.Fatalf("returned color should be valid, got error: %v", err)
		}

		seen[color.String()] = struct{}{}
	}

	if len(seen) == 0 {
		t.Fatalf("expected at least one color to be picked")
	}
}
