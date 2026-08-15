package cache

import (
	"testing"
	"time"
)

func TestFingerprintDeterministicForStructuredInputs(t *testing.T) {
	input := struct {
		Name  string            `json:"name"`
		Tags  []string          `json:"tags"`
		When  time.Time         `json:"when"`
		Attrs map[string]string `json:"attrs"`
	}{
		Name:  "repo",
		Tags:  []string{"one", "two"},
		When:  time.Unix(123, 0).UTC(),
		Attrs: map[string]string{"z": "last", "a": "first"},
	}

	first, err := Fingerprint(input)
	if err != nil {
		t.Fatalf("Fingerprint: %v", err)
	}
	second, err := Fingerprint(input)
	if err != nil {
		t.Fatalf("Fingerprint second call: %v", err)
	}
	if first != second {
		t.Fatalf("identical inputs produced different fingerprints: %q != %q", first, second)
	}

	changed := input
	changed.Tags = []string{"one", "changed"}
	third, err := Fingerprint(changed)
	if err != nil {
		t.Fatalf("Fingerprint changed input: %v", err)
	}
	if first == third {
		t.Fatal("changed structured input reused the original fingerprint")
	}
}

func TestFingerprintPreservesPartBoundaries(t *testing.T) {
	first, err := Fingerprint("ab", "c")
	if err != nil {
		t.Fatalf("Fingerprint first: %v", err)
	}
	second, err := Fingerprint("a", "bc")
	if err != nil {
		t.Fatalf("Fingerprint second: %v", err)
	}
	if first == second {
		t.Fatal("different fingerprint parts collided")
	}
}

func TestFingerprintRejectsUnsupportedValues(t *testing.T) {
	if _, err := Fingerprint(func() {}); err == nil {
		t.Fatal("expected unsupported value to return an error")
	}
}
