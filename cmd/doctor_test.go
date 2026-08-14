package cmd

import (
	"strings"
	"testing"
)

func TestDoctorFailureIsReturnedAfterSummary(t *testing.T) {
	err := doctorFailureError(1)
	if err == nil {
		t.Fatal("expected doctor failure error")
	}
	if !strings.Contains(err.Error(), "failed check") {
		t.Fatalf("unexpected diagnostic error: %v", err)
	}
}

func TestDoctorFailureIsNilWhenChecksPass(t *testing.T) {
	if err := doctorFailureError(0); err != nil {
		t.Fatalf("expected nil error when checks pass, got %v", err)
	}
}
