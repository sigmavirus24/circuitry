package dynamodb

import "fmt"

// OperationType is used to quickly identify the kind of operation the backend
// is performing
type OperationType int

const (
	// OpGetItem represents the GetItem Operation
	OpGetItem OperationType = iota
	// OpUpdateItem represents the UpdateItem Operation
	OpUpdateItem
	// OpAcquireLock represents the AcquireLock Operation
	OpAcquireLock
	// OpReleaseLock represents the ReleaseLock Operation
	OpReleaseLock
	// OpCreateTable represents the CreateTable Operation
	OpCreateTable
)

func (t OperationType) String() string {
	switch t {
	case OpGetItem:
		return "GetItem"
	case OpUpdateItem:
		return "UpdateItem"
	case OpAcquireLock:
		return "AcquireLock"
	case OpReleaseLock:
		return "ReleaseLock"
	case OpCreateTable:
		return "CreateTable"
	default:
		return "unknown-operation"
	}
}

// RemoteBackendError wraps errors from aws-sdk-v2 and the DynamoDB Client
type RemoteBackendError struct {
	Err       error
	TableName string
	Operation OperationType
}

func (e *RemoteBackendError) Unwrap() error {
	return e.Err
}

func (e *RemoteBackendError) Error() string {
	return fmt.Sprintf("dynamodb backend could not perform %s on %q: %s", e.Operation, e.TableName, e.Err)
}

var _ error = (*RemoteBackendError)(nil)

// LocalBackendError wraps errors operating locally before speaking to the
// DynamoDB server
type LocalBackendError struct {
	Err     error
	Message string
}

func (e *LocalBackendError) Error() string {
	return fmt.Sprintf("%s: %s", e.Message, e.Err)
}

func (e *LocalBackendError) Unwrap() error {
	return e.Err
}

var _ error = (*LocalBackendError)(nil)
