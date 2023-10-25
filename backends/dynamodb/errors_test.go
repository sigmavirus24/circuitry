package dynamodb_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sigmavirus24/circuitry/backends/dynamodb"
)

func TestOperationTypeString(t *testing.T) {
	testCases := map[string]struct {
		opType   dynamodb.OperationType
		expected string
	}{
		"OpGetItem":     {dynamodb.OpGetItem, "GetItem"},
		"OpUpdateItem":  {dynamodb.OpUpdateItem, "UpdateItem"},
		"OpAcquireLock": {dynamodb.OpAcquireLock, "AcquireLock"},
		"OpReleaseLock": {dynamodb.OpReleaseLock, "ReleaseLock"},
		"OpCreateTable": {dynamodb.OpCreateTable, "CreateTable"},
		"OpUnknown":     {dynamodb.OpCreateTable + dynamodb.OpAcquireLock, "unknown-operation"},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			if actual := tc.opType.String(); actual != tc.expected {
				t.Fatalf("expected OperationType(%d).String() = %q, got %q", tc.opType, tc.expected, actual)
			}
		})
	}
}

func TestRemoteBackendError(t *testing.T) {
	actualErr := errors.New("test")
	err := &dynamodb.RemoteBackendError{
		Err:       actualErr,
		TableName: "fake-table-name",
		Operation: dynamodb.OpGetItem,
	}
	if !errors.Is(err, actualErr) {
		t.Fatal("RemoteBackendError did not unwrap to underlying error")
	}
	expected := "dynamodb backend could not perform GetItem on \"fake-table-name\": test"
	if actual := err.Error(); actual != expected {
		t.Fatalf("expected RemoteBackendError.Error() = %q; got %q", expected, actual)
	}
}

func TestLocalBackendError(t *testing.T) {
	actualErr := errors.New("test")
	message := "fake message"
	err := &dynamodb.LocalBackendError{actualErr, message}
	if !errors.Is(err, actualErr) {
		t.Fatal("LocalBackendError did not unwrap to underlying error")
	}
	expected := fmt.Sprintf("%s: %s", message, actualErr)
	if actual := err.Error(); actual != expected {
		t.Fatalf("expected LocalBackendError.Error() = %q; got %q", expected, actual)
	}
}
