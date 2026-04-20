package slug

import (
	"strings"
	"testing"
)

func TestGenerateFileSlug(t *testing.T) {
	s, err := GenerateFileSlug()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s, "f~") {
		t.Errorf("expected f~ prefix, got %s", s)
	}
	if len(s) != 12 {
		t.Errorf("expected length 12, got %d", len(s))
	}
}

func TestGenerateShareSlug(t *testing.T) {
	s, err := GenerateShareSlug()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s, "s~") {
		t.Errorf("expected s~ prefix, got %s", s)
	}
	if len(s) != 12 {
		t.Errorf("expected length 12, got %d", len(s))
	}
}

func TestSlugUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		s, _ := GenerateFileSlug()
		if seen[s] {
			t.Fatalf("duplicate slug: %s", s)
		}
		seen[s] = true
	}
}