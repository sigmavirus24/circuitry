package dynamodb

import (
	"context"
	"fmt"
	"sync"

	ddblock "cirello.io/dynamolock/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	ddb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sigmavirus24/circuitry"
)

// DynamoClient defines the necessary attributes for a DynamoDB client for the
// purpose of testing and mocks
type DynamoClient interface {
	GetItem(ctx context.Context, params *ddb.GetItemInput, optFns ...func(*ddb.Options)) (*ddb.GetItemOutput, error)
	PutItem(ctx context.Context, params *ddb.PutItemInput, optFns ...func(*ddb.Options)) (*ddb.PutItemOutput, error)
	UpdateItem(ctx context.Context, params *ddb.UpdateItemInput, optFns ...func(*ddb.Options)) (*ddb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, params *ddb.DeleteItemInput, optFns ...func(*ddb.Options)) (*ddb.DeleteItemOutput, error)
	CreateTable(ctx context.Context, params *ddb.CreateTableInput, optFns ...func(*ddb.Options)) (*ddb.CreateTableOutput, error)
}

// DynamoLocker defines the necessary attributes for a DynamoDB Lock
// implementation
type DynamoLocker interface {
	AcquireLockWithContext(ctx context.Context, key string, opts ...ddblock.AcquireLockOption) (*ddblock.Lock, error)
	CreateTableWithContext(ctx context.Context, tableName string, opts ...ddblock.CreateTableOption) (*ddb.CreateTableOutput, error)
	ReleaseLockWithContext(ctx context.Context, lockItem *ddblock.Lock, opts ...ddblock.ReleaseLockOption) (bool, error)
}

// DynamoLock wraps a DynamoDB Lock from cirello.io/dynamolock/v2 in a
// sync.Locker interface
type DynamoLock struct {
	ddbClient DynamoClient
	lock      *ddblock.Lock
}

// Lock does nothing as the lock has alrady been acquired at this point
func (l *DynamoLock) Lock() {}

// Unlock releases the backend lock
func (l *DynamoLock) Unlock() {
	l.lock.Close()
}

var _ sync.Locker = (*DynamoLock)(nil)

// Backend provides the StorageBackender interface wrapper for DynamoDB
// support
type Backend struct {
	Client                          DynamoClient
	LockClient                      DynamoLocker
	CircuitTableName, LockTableName string
	AcquireLockOpts                 []ddblock.AcquireLockOption
	ReleaseLockOpts                 []ddblock.ReleaseLockOption
}

// Store CircuitInformation in DynamoDB in the specified CircuitTableName
// table
func (b *Backend) Store(ctx context.Context, name string, ci circuitry.CircuitInformation) error {
	record := recordFromCircuitInformation(ci)
	record.Name = name
	expr, err := record.ToUpdateExpression()
	if err != nil {
		return err
	}
	key, err := attributevalue.Marshal(record.Name)
	if err != nil {
		return &LocalBackendError{Err: err, Message: fmt.Sprintf("could not marshal %q with AWS SDK for dynamodb hash key", name)}
	}
	_, err = b.Client.UpdateItem(ctx, &ddb.UpdateItemInput{
		TableName:                 aws.String(b.CircuitTableName),
		Key:                       map[string]ddbtypes.AttributeValue{KeyName: key},
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
		ReturnValues:              ddbtypes.ReturnValueNone,
	})
	if err != nil {
		return &RemoteBackendError{Err: err, Operation: OpUpdateItem, TableName: b.CircuitTableName}
	}
	return nil
}

// Retrieve CircuitInformation in DynamoDB from the specified CircuitTableName
// table
func (b *Backend) Retrieve(ctx context.Context, name string) (circuitry.CircuitInformation, error) {
	record := circuitInfoRecord{Name: name}
	keyValue, err := attributevalue.Marshal(record.Name)
	if err != nil {
		return circuitry.CircuitInformation{}, &LocalBackendError{Err: err, Message: fmt.Sprintf("cannot marshal %q with AWS SDK for dynamodb hash key", name)}
	}
	getItem := &ddb.GetItemInput{
		Key:       map[string]ddbtypes.AttributeValue{KeyName: keyValue},
		TableName: aws.String(b.CircuitTableName),
	}
	response, err := b.Client.GetItem(ctx, getItem)
	if err != nil {
		return circuitry.CircuitInformation{}, &RemoteBackendError{Err: err, Operation: OpGetItem, TableName: b.CircuitTableName}
	}
	err = attributevalue.UnmarshalMap(response.Item, &record)
	if err != nil {
		return circuitry.CircuitInformation{}, &LocalBackendError{Err: err, Message: fmt.Sprintf("cannot unmarshal data for %q", name)}
	}
	return record.ToCircuitInformation(), nil
}

// Lock created in DynamoDB using the LockTableName table
func (b *Backend) Lock(ctx context.Context, name string) (sync.Locker, error) {
	lock, err := b.LockClient.AcquireLockWithContext(ctx, name, ddblock.FailIfLocked(), ddblock.WithDeleteLockOnRelease())
	if err != nil {
		return nil, &RemoteBackendError{Err: err, Operation: OpAcquireLock, TableName: b.LockTableName}
	}
	return &DynamoLock{b.Client, lock}, nil
}

var _ circuitry.StorageBackender = (*Backend)(nil)

// WithDynamoBackend can be used to configure a circuitry.FactorySettings
// object to use DynamoDB as the backend.
func WithDynamoBackend(client DynamoClient, locker DynamoLocker, circuitInformationTableName, lockTableName string, lockOpts ...ddblock.ClientOption) circuitry.SettingsOption {
	return func(s *circuitry.FactorySettings) error {
		if s.StorageBackend != nil {
			return circuitry.ErrStorageBackendAlreadySet
		}
		if locker == nil {
			var err error
			locker, err = ddblock.New(client, lockTableName, lockOpts...)
			if err != nil {
				return circuitry.ErrProvisioningStorageBackend
			}
		}
		backend := Backend{
			Client:           client,
			LockClient:       locker,
			CircuitTableName: circuitInformationTableName,
			LockTableName:    lockTableName,
		}
		s.StorageBackend = &backend
		return nil
	}
}
