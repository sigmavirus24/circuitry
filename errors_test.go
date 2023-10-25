package circuitry_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sigmavirus24/circuitry"
)

func TestExpectedConditionError(t *testing.T) {
	originalErr := fmt.Errorf("test error")
	err := circuitry.WrapExpectedConditionError(originalErr)
	if s := err.Error(); s != originalErr.Error() {
		t.Fatalf("expected err.Error() = %s; got %s", originalErr.Error(), s)
	}
	if !errors.Is(err, originalErr) {
		t.Fatalf("expected err to unwrap to originalErr but it didn't")
	}
	if e, ok := err.(circuitry.IsExpectedErrorer); ok && !e.IsExpected() {
		t.Fatalf("expected ExpectedConditionError.IsExpected() = true, but got false")
	}
}
