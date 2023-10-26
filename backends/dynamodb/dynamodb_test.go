package dynamodb_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	ddblock "cirello.io/dynamolock/v2"
	ddb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sigmavirus24/circuitry"
	"github.com/sigmavirus24/circuitry/backends/dynamodb"
	ddbbackend "github.com/sigmavirus24/circuitry/backends/dynamodb"
)

type ddbLockerMock struct {
	acquireLockOptions      []ddblock.AcquireLockOption
	keys                    []string
	acquireLockReturnErrors []error
}

func (l *ddbLockerMock) AcquireLockWithContext(_ context.Context, key string, opts ...ddblock.AcquireLockOption) (*ddblock.Lock, error) {
	l.keys = append(l.keys, key)
	index := len(l.keys) - 1
	if len(l.acquireLockReturnErrors) > index {
		return nil, l.acquireLockReturnErrors[index]
	}
	return &ddblock.Lock{}, nil
}
func (l *ddbLockerMock) CreateTableWithContext(_ context.Context, tableName string, opts ...ddblock.CreateTableOption) (*ddb.CreateTableOutput, error) {
	return nil, nil
}
func (l *ddbLockerMock) ReleaseLockWithContext(_ context.Context, lockItem *ddblock.Lock, opts ...ddblock.ReleaseLockOption) (bool, error) {
	return false, nil
}

func newDDBLockerMock() *ddbLockerMock {
	return &ddbLockerMock{
		acquireLockOptions:      make([]ddblock.AcquireLockOption, 0),
		keys:                    make([]string, 0),
		acquireLockReturnErrors: make([]error, 0),
	}
}

type ddbMock struct {
	getItemInputs           []*ddb.GetItemInput
	getItemOutputs          []*ddb.GetItemOutput
	getItemErrors           []error
	getItemOutputCounter    uint
	updateItemInputs        []*ddb.UpdateItemInput
	updateItemOutputs       []*ddb.UpdateItemOutput
	updateItemErrors        []error
	updateItemOutputCounter uint
}

func newDDBMock() *ddbMock {
	return &ddbMock{
		getItemInputs:     make([]*ddb.GetItemInput, 0),
		getItemOutputs:    make([]*ddb.GetItemOutput, 0),
		getItemErrors:     make([]error, 0),
		updateItemInputs:  make([]*ddb.UpdateItemInput, 0),
		updateItemOutputs: make([]*ddb.UpdateItemOutput, 0),
		updateItemErrors:  make([]error, 0),
	}
}

func (m *ddbMock) AddGetItemError(err error) {
	m.getItemErrors = append(m.getItemErrors, err)
}

func (m *ddbMock) AddGetItemOutput(out *ddb.GetItemOutput) {
	m.getItemOutputs = append(m.getItemOutputs, out)
}

func (m *ddbMock) AddUpdateItemError(err error) {
	m.updateItemErrors = append(m.updateItemErrors, err)
}

func (m *ddbMock) AddUpdateItemOutput(out *ddb.UpdateItemOutput) {
	m.updateItemOutputs = append(m.updateItemOutputs, out)
}

func (m *ddbMock) GetItem(_ context.Context, params *ddb.GetItemInput, optFns ...func(*ddb.Options)) (*ddb.GetItemOutput, error) {
	m.getItemInputs = append(m.getItemInputs, params)
	if outputs := uint(len(m.getItemOutputs)); m.getItemOutputCounter >= outputs {
		errorIndex := m.getItemOutputCounter - outputs
		m.getItemOutputCounter++
		return nil, m.getItemErrors[errorIndex]
	}
	output := m.getItemOutputs[m.getItemOutputCounter]
	m.getItemOutputCounter++
	return output, nil
}

func (m *ddbMock) PutItem(_ context.Context, params *ddb.PutItemInput, optFns ...func(*ddb.Options)) (*ddb.PutItemOutput, error) {
	return nil, nil
}

func (m *ddbMock) UpdateItem(_ context.Context, params *ddb.UpdateItemInput, optFns ...func(*ddb.Options)) (*ddb.UpdateItemOutput, error) {
	m.updateItemInputs = append(m.updateItemInputs, params)
	if outputs := uint(len(m.updateItemOutputs)); m.updateItemOutputCounter >= outputs {
		errorIndex := m.updateItemOutputCounter - outputs
		m.updateItemOutputCounter++
		return nil, m.updateItemErrors[errorIndex]
	}
	output := m.updateItemOutputs[m.updateItemOutputCounter]
	m.updateItemOutputCounter++
	return output, nil
}

func (m *ddbMock) DeleteItem(_ context.Context, params *ddb.DeleteItemInput, optFns ...func(*ddb.Options)) (*ddb.DeleteItemOutput, error) {
	return nil, nil
}

func (m *ddbMock) CreateTable(_ context.Context, params *ddb.CreateTableInput, optFns ...func(*ddb.Options)) (*ddb.CreateTableOutput, error) {
	return nil, nil
}

var _ ddbbackend.DynamoClient = (*ddbMock)(nil)

func TestWithDynamoBackend(t *testing.T) {
	client := newDDBMock()
	opt := ddbbackend.WithDynamoBackend(client, nil, "circuit_info", "circuit_breaker_locks")
	s, err := circuitry.NewFactorySettings(opt)
	if err != nil {
		t.Fatalf("expected successful settings creation; got %v", err)
	}
	if _, ok := s.StorageBackend.(*ddbbackend.Backend); !ok {
		t.Fatal("expected configured StorageBackend to be dynamodb backend; but it wasn't")
	}

	_, err = circuitry.NewFactorySettings(opt, opt)
	if !errors.Is(err, circuitry.ErrStorageBackendAlreadySet) {
		t.Fatalf("expected err = circuitry.ErrStorageBackendAlreadySet; got %v", err)
	}

	opt = ddbbackend.WithDynamoBackend(client, nil, "circuit_info", "circuit_breaker_locks", ddblock.WithHeartbeatPeriod(5*time.Second), ddblock.WithLeaseDuration(5*time.Second))
	_, err = circuitry.NewFactorySettings(opt)
	if !errors.Is(err, circuitry.ErrProvisioningStorageBackend) {
		t.Fatalf("expected to get an ErrProvisiongStorageBackend from dynamolock library, but got err = %v", err)
	}
}

func intAttrValueMember(i uint64) *ddbtypes.AttributeValueMemberN {
	return &ddbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", i)}
}

func strAttrValueMember(v string) *ddbtypes.AttributeValueMemberS {
	return &ddbtypes.AttributeValueMemberS{Value: v}
}

func ciToAVMap(ci circuitry.CircuitInformation) map[string]ddbtypes.AttributeValue {
	return map[string]ddbtypes.AttributeValue{
		"generation":            intAttrValueMember(ci.Generation),
		"consecutive_successes": intAttrValueMember(ci.ConsecutiveSuccesses),
		"consecutive_failures":  intAttrValueMember(ci.ConsecutiveFailures),
		"total_successes":       intAttrValueMember(ci.TotalSuccesses),
		"total_failures":        intAttrValueMember(ci.TotalFailures),
		"total":                 intAttrValueMember(ci.Total),
		"state":                 intAttrValueMember(uint64(ci.State)),
		"expires_after":         strAttrValueMember(ci.ExpiresAfter.Format("2006-01-02T15:04:05Z07:00")),
	}
}

func TestBackendRetrieve(t *testing.T) {
	client := newDDBMock()
	expected := circuitry.CircuitInformation{
		Generation:           2,
		State:                circuitry.CircuitOpen,
		ConsecutiveFailures:  10,
		ConsecutiveSuccesses: 0,
		Total:                20,
		TotalFailures:        15,
		TotalSuccesses:       5,
		ExpiresAfter:         time.Now().Add(time.Hour).Truncate(time.Second),
	}
	client.AddGetItemOutput(&ddb.GetItemOutput{
		Item: ciToAVMap(expected),
	})
	lockClient := newDDBLockerMock()
	backend := ddbbackend.Backend{
		Client:           client,
		LockClient:       lockClient,
		CircuitTableName: "circuit_information",
		LockTableName:    "circuit_locks",
	}
	actual, err := backend.Retrieve(context.TODO(), "circuit-name-retrieve")
	if err != nil {
		t.Fatalf("expected to retrieve information, got err = %v", err)
	}
	deepEqCi(t, expected, actual)
}

func TestBackendRetrieveUnmarshalError(t *testing.T) {
	client := newDDBMock()
	expected := circuitry.CircuitInformation{
		Generation:           2,
		State:                circuitry.CircuitOpen,
		ConsecutiveFailures:  10,
		ConsecutiveSuccesses: 0,
		Total:                20,
		TotalFailures:        15,
		TotalSuccesses:       5,
		ExpiresAfter:         time.Now().Add(time.Hour).Truncate(time.Second),
	}
	ciAVMap := ciToAVMap(expected)
	ciAVMap["total"] = strAttrValueMember("1")
	client.AddGetItemOutput(&ddb.GetItemOutput{
		Item: ciAVMap,
	})
	lockClient := newDDBLockerMock()
	backend := ddbbackend.Backend{
		Client:           client,
		LockClient:       lockClient,
		CircuitTableName: "circuit_information",
		LockTableName:    "circuit_locks",
	}
	_, err := backend.Retrieve(context.TODO(), "circuit-name-retrieve")
	var localErr *dynamodb.LocalBackendError
	if !errors.As(err, &localErr) {
		t.Fatalf("expected to receive LocalBackendError, got err = %v", err)
	}
}

func TestBackendRetrieveNotFound(t *testing.T) {
	client := newDDBMock()
	client.AddGetItemOutput(&ddb.GetItemOutput{Item: map[string]ddbtypes.AttributeValue{}})

	lockClient := newDDBLockerMock()
	backend := ddbbackend.Backend{
		Client:           client,
		LockClient:       lockClient,
		CircuitTableName: "circuit_information",
		LockTableName:    "circuit_locks",
	}
	ci, err := backend.Retrieve(context.TODO(), "circuit-name-not-found")
	if err != nil {
		t.Fatalf("expected to retrieve information, got err = %v", err)
	}
	if ci != (circuitry.CircuitInformation{}) {
		t.Fatal("expected to get nil information, got non-nil CircuitInformation")
	}
}

func TestBackendRetrieveWithError(t *testing.T) {
	client := newDDBMock()
	ddbErr := errors.New("test")
	client.AddGetItemError(ddbErr)

	lockClient := newDDBLockerMock()
	backend := ddbbackend.Backend{
		Client:           client,
		LockClient:       lockClient,
		CircuitTableName: "circuit_information_retrieve_error",
		LockTableName:    "circuit_locks_retrieve_error",
	}

	_, err := backend.Retrieve(context.TODO(), "circuit-name-returns-error")
	if !errors.Is(err, ddbErr) {
		t.Fatalf("expected to get test error, instead got %v", err)
	}

}

func TestBackendStoreNoTable(t *testing.T) {
	client := newDDBMock()
	client.AddUpdateItemError(&ddbtypes.ResourceNotFoundException{})

	lockClient := newDDBLockerMock()
	backend := ddbbackend.Backend{
		Client:           client,
		LockClient:       lockClient,
		CircuitTableName: "circuit_information_store_no_table",
		LockTableName:    "circuit_locsk_store_no_table",
	}
	err := backend.Store(context.TODO(), "circuit-name-does-not-matter", circuitry.CircuitInformation{})
	var notFound *ddbtypes.ResourceNotFoundException
	if !errors.As(err, &notFound) {
		t.Fatalf("expected err to unwrap to ResourceNotFound; got err = %T(%v)", err, err)
	}
}

func TestBackendStore(t *testing.T) {
	client := newDDBMock()
	client.AddUpdateItemOutput(&ddb.UpdateItemOutput{})

	lockClient := newDDBLockerMock()
	backend := ddbbackend.Backend{
		Client:           client,
		LockClient:       lockClient,
		CircuitTableName: "circuit_information_store_no_table",
		LockTableName:    "circuit_locsk_store_no_table",
	}
	err := backend.Store(context.TODO(), "circuit-name-does-not-matter", circuitry.CircuitInformation{})
	if err != nil {
		t.Fatalf("expected err to be nil; got err = %T(%v)", err, err)
	}
}

func TestBackendLock(t *testing.T) {
	client := newDDBMock()
	lockClient := newDDBLockerMock()
	lockErr := errors.New("test")
	lockClient.acquireLockReturnErrors = append(lockClient.acquireLockReturnErrors, lockErr)

	backend := ddbbackend.Backend{
		Client:           client,
		LockClient:       lockClient,
		CircuitTableName: "circuit_information_store_no_table",
		LockTableName:    "circuit_locsk_store_no_table",
	}
	_, err := backend.Lock(context.TODO(), "fake-key-lock-error")
	if !errors.Is(err, lockErr) {
		t.Fatalf("expected to get lockErr trying to lock resource; got err = %v", err)
	}

	lock, err := backend.Lock(context.TODO(), "fake-key")
	if err != nil {
		t.Fatalf("expected to get a lock, but got err = %v", err)
	}
	lock.Lock()
	defer lock.Unlock()
}
