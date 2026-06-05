package presets

import (
	"slices"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name      string
		preset    string
		wantCats  []string
		wantErr   bool
	}{
		{
			name:     "quick preset",
			preset:   Quick,
			wantCats: []string{CatConfig},
		},
		{
			name:     "full preset",
			preset:   Full,
			wantCats: AllCategories,
		},
		{
			name:     "skills preset",
			preset:   Skills,
			wantCats: []string{CatSkills},
		},
		{
			name:    "unknown preset",
			preset:  "bananas",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.preset)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.wantCats) {
				t.Errorf("categories = %v, want %v", got, tt.wantCats)
			}
		})
	}
}

func TestResolve_ReturnsCopy(t *testing.T) {
	cats1, _ := Resolve(Quick)
	cats2, _ := Resolve(Quick)

	// Mutate one, verify the other is unchanged.
	cats1[0] = "corrupted"

	if cats2[0] != CatConfig {
		t.Errorf("Resolve did not return a copy: cats2[0] = %q after mutating cats1", cats2[0])
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != 3 {
		t.Errorf("Names() len = %d, want 3", len(names))
	}
	expected := []string{"quick", "full", "skills"}
	if !slices.Equal(names, expected) {
		t.Errorf("Names() = %v, want %v", names, expected)
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{Quick, true},
		{Full, true},
		{Skills, true},
		{"all", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValid(tt.name)
			if got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
