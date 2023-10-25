package circuitry

import (
	"strings"
	"testing"
)

func TestSettingsConflictError(t *testing.T) {
	e := newSettingsConflictError("Name")
	if !strings.HasPrefix(e.Error(), "Name ") {
		t.Fatal("expected CircuitSpecificSettingsConflictError to fallback to default formatting but it didn't")
	}

	e = SettingsConflictError{"Name", ""}
	if !strings.HasPrefix(e.Error(), "Name ") {
		t.Fatal("expected CircuitSpecificSettingsConflictError to fallback to default formatting but it didn't")
	}
}

func TestCircuitSpecificSettingsConflictError(t *testing.T) {
	e := newCircuitSpecificSettingsConflictError("Name", "Circuit Name")
	if !strings.HasPrefix(e.Error(), "Name ") {
		t.Fatal("expected CircuitSpecificSettingsConflictError to fallback to default formatting but it didn't")
	}

	e = CircuitSpecificSettingsConflictError{SettingsConflictError{"Name", ""}, "Circuit Name"}
	if !strings.HasPrefix(e.Error(), "Name ") {
		t.Fatal("expected CircuitSpecificSettingsConflictError to fallback to default formatting but it didn't")
	}
}
