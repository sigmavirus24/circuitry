package dynamodb_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	ddblock "cirello.io/dynamolock/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	tc "github.com/testcontainers/testcontainers-go"
	tcw "github.com/testcontainers/testcontainers-go/wait"

	"github.com/sigmavirus24/circuitry"
	ddbbackend "github.com/sigmavirus24/circuitry/backends/dynamodb"
)

const envVar string = "DYNAMODB_URL"

var dynamoDBUrl string

func init() {
	if dbURL := os.Getenv(envVar); dbURL != "" {
		dynamoDBUrl = dbURL
	}
}

func dynamodbContainer(ctx context.Context) (tc.Container, func(*testing.T), error) {
	req := tc.GenericContainerRequest{
		ProviderType: tc.ProviderPodman,
		ContainerRequest: tc.ContainerRequest{
			Image:        "docker.io/amazon/dynamodb-local",
			ExposedPorts: []string{"8001/tcp"},
			WaitingFor:   tcw.ForExposedPort(),
			Cmd:          []string{"-jar", "DynamoDBLocal.jar", "-inMemory"},
		},
		Started: true,
	}
	ddbContainer, err := tc.GenericContainer(ctx, req)
	return ddbContainer, func(t *testing.T) {
		if ddbContainer == nil {
			return
		}
		if err := ddbContainer.Terminate(ctx); err != nil {
			t.Fatalf("could not terminate dynamodb container; %v", err)
		}
	}, err
}

func dynamodbClientFromContainer(ddbContainer tc.Container) (*dynamodb.Client, error) {
	endpoint, err := ddbContainer.Endpoint(context.TODO(), "")
	if err != nil {
		return nil, err
	}
	awsCfg := aws.NewConfig()
	awsCfg.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{URL: endpoint}, nil
	})
	return dynamodb.NewFromConfig(*awsCfg), nil
}

func dynamodbClientFromURL() *dynamodb.Client {
	awsCfg := aws.NewConfig()
	awsCfg.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{URL: dynamoDBUrl}, nil
	})
	keyID := awsCred()
	secret := awsCred()
	token := awsCred()
	awsCfg.Credentials = aws.CredentialsProviderFunc(func(_ context.Context) (aws.Credentials, error) {
		return aws.Credentials{
			AccessKeyID:     keyID,
			SecretAccessKey: secret,
			SessionToken:    token,
			Source:          "environment",
			CanExpire:       false,
		}, nil
	})
	return dynamodb.NewFromConfig(*awsCfg)
}

func maybeSkip(t *testing.T) {
	t.Helper()

	if dynamoDBUrl == "" {
		t.Skipf("No %s environment variable specified, skipping test", envVar)
	}
}

func awsCred() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

func TestBackendIntegrationRetrieveNoTable(t *testing.T) {
	maybeSkip(t)
	t.Parallel()
	/* Re-enable when testcontainers + podman works better
	ddbContainer, termFn, err := dynamodbContainer(context.TODO())
	defer termFn(t)
	if err != nil {
		t.Fatalf("expected to be able to launch a new container; got %v", err)
	}
	*/

	ddbClient := dynamodbClientFromURL()

	testID := uuid.NewString()
	circuitTable := fmt.Sprintf("circuit_info_%s", testID)
	locksTable := fmt.Sprintf("circuit_breaker_locks_%s", testID)
	lockClient, err := ddblock.New(ddbClient, locksTable)
	if err != nil {
		t.Fatalf("could not create dynamodb lock client: %v", err)
	}

	backend := ddbbackend.Backend{
		Client:           ddbClient,
		LockClient:       lockClient,
		CircuitTableName: circuitTable,
		LockTableName:    locksTable,
	}

	_, err = backend.Retrieve(context.TODO(), "fake-key")
	be, ok := err.(*ddbbackend.RemoteBackendError)
	if !ok {
		t.Fatalf("expected to receive a wrapped error that no table existed, got %v", err)
	}
	if be.Operation != ddbbackend.OpGetItem || be.TableName != circuitTable {
		t.Fatalf("expected BackendError{Operation: OpGetItem, TableName: %q}, got %+v", circuitTable, be)
	}
	var notFound *types.ResourceNotFoundException
	if !errors.As(be.Unwrap(), &notFound) {
		t.Fatalf("expected to Unwrap() backend error to aws ResourceNotFoundException, got %T", be.Unwrap())
	}
}

func TestBackendIntegrationRetrieveFromEmptyTable(t *testing.T) {
	maybeSkip(t)
	t.Parallel()

	ddbClient := dynamodbClientFromURL()

	testID := uuid.NewString()
	circuitTable := fmt.Sprintf("circuit_info_%s", testID)
	locksTable := fmt.Sprintf("circuit_breaker_locks_%s", testID)
	lockClient, err := ddblock.New(ddbClient, locksTable)
	if err != nil {
		t.Fatalf("could not create dynamodb lock client: %v", err)
	}
	backend := ddbbackend.Backend{
		Client:           ddbClient,
		LockClient:       lockClient,
		CircuitTableName: circuitTable,
		LockTableName:    locksTable,
	}

	_, err = ddbbackend.CreateCircuitInformationTable(context.TODO(), ddbClient, circuitTable)
	if err != nil {
		t.Fatalf("failed to create new test table: %v", err)
	}
	defer func() {
		ddbClient.DeleteTable(context.TODO(), &dynamodb.DeleteTableInput{
			TableName: aws.String(circuitTable),
		})
	}()

	_, err = backend.Retrieve(context.TODO(), "fake-key")
	if err != nil {
		t.Fatalf("expected to get empty CircuitInformation, got err = %v", err)
	}
}

func TestBackendIntegrationRetrieveRealData(t *testing.T) {
	maybeSkip(t)
	t.Parallel()

	ddbClient := dynamodbClientFromURL()
	testID := uuid.NewString()
	expected := circuitry.CircuitInformation{
		Generation:           1,
		State:                circuitry.CircuitClosed,
		ConsecutiveSuccesses: 5,
		Total:                5,
		TotalSuccesses:       5,
		ExpiresAfter:         time.Now().Add(time.Hour).Truncate(time.Second),
	}
	item := ciToAVMap(expected)
	key := fmt.Sprintf("circuit-breaker-%s", testID)
	breakerKey, err := attributevalue.Marshal(key)
	if err != nil {
		t.Fatalf("cannot marshal key, got %v", err)
	}
	item[ddbbackend.KeyName] = breakerKey

	circuitTable := fmt.Sprintf("circuit_info_%s", testID)
	locksTable := fmt.Sprintf("circuit_breaker_locks_%s", testID)
	lockClient, err := ddblock.New(ddbClient, locksTable)
	if err != nil {
		t.Fatalf("could not create dynamodb lock client: %v", err)
	}
	backend := ddbbackend.Backend{
		Client:           ddbClient,
		LockClient:       lockClient,
		CircuitTableName: circuitTable,
		LockTableName:    locksTable,
	}

	_, err = ddbbackend.CreateCircuitInformationTable(context.TODO(), ddbClient, circuitTable)
	if err != nil {
		t.Fatalf("failed to create new test table: %v", err)
	}
	defer func() {
		ddbClient.DeleteTable(context.TODO(), &dynamodb.DeleteTableInput{
			TableName: aws.String(circuitTable),
		})
	}()

	_, err = ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(circuitTable),
		Item:      item,
	})
	if err != nil {
		t.Fatalf("couldn't pre-populate test data, got err = %v", err)
	}

	actual, err := backend.Retrieve(context.TODO(), key)
	if err != nil {
		t.Fatalf("expected to get empty CircuitInformation, got err = %v", err)
	}

	if expected != actual {
		t.Fatalf("expected to get %+v; got %+v", expected, actual)
	}
}

func TestBackendIntegrationStoreNoTable(t *testing.T) {
	maybeSkip(t)
	t.Parallel()

	ddbClient := dynamodbClientFromURL()
	testID := uuid.NewString()
	expected := circuitry.CircuitInformation{
		Generation:           1,
		State:                circuitry.CircuitClosed,
		ConsecutiveSuccesses: 5,
		Total:                5,
		TotalSuccesses:       5,
		ExpiresAfter:         time.Now().Add(time.Hour).Truncate(time.Second),
	}
	key := fmt.Sprintf("circuit-breaker-%s", testID)

	circuitTable := fmt.Sprintf("circuit_info_%s", testID)
	locksTable := fmt.Sprintf("circuit_breaker_locks_%s", testID)
	lockClient, err := ddblock.New(ddbClient, locksTable)
	if err != nil {
		t.Fatalf("could not create dynamodb lock client: %v", err)
	}
	backend := ddbbackend.Backend{
		Client:           ddbClient,
		LockClient:       lockClient,
		CircuitTableName: circuitTable,
		LockTableName:    locksTable,
	}

	if err != nil {
		t.Fatalf("couldn't pre-populate test data, got err = %v", err)
	}

	err = backend.Store(context.TODO(), key, expected)
	var be *ddbbackend.RemoteBackendError
	if !errors.As(err, &be) {
		t.Fatalf("expected to get RemoteBackendError; got err = %T(%v)", err, err)
	}
	if be.Operation != ddbbackend.OpUpdateItem {
		t.Fatalf("expected RemoteBackendError.Operation = OpUpdateItem; got Op%s", be.Operation)
	}
	var notFound *types.ResourceNotFoundException
	if !errors.As(be.Unwrap(), &notFound) {
		t.Fatalf("expected to Unwrap() backend error to aws ResourceNotFoundException, got %T", be.Unwrap())
	}
}

func TestBackendIntegrationStoreRealData(t *testing.T) {
	maybeSkip(t)
	t.Parallel()

	ddbClient := dynamodbClientFromURL()
	testID := uuid.NewString()
	expected := circuitry.CircuitInformation{
		Generation:           1,
		State:                circuitry.CircuitClosed,
		ConsecutiveSuccesses: 5,
		Total:                5,
		TotalSuccesses:       5,
		ExpiresAfter:         time.Now().Add(time.Hour).Truncate(time.Second),
	}
	key := fmt.Sprintf("circuit-breaker-%s", testID)

	circuitTable := fmt.Sprintf("circuit_info_%s", testID)
	locksTable := fmt.Sprintf("circuit_breaker_locks_%s", testID)
	lockClient, err := ddblock.New(ddbClient, locksTable)
	if err != nil {
		t.Fatalf("could not create dynamodb lock client: %v", err)
	}
	backend := ddbbackend.Backend{
		Client:           ddbClient,
		LockClient:       lockClient,
		CircuitTableName: circuitTable,
		LockTableName:    locksTable,
	}

	_, err = ddbbackend.CreateCircuitInformationTable(context.TODO(), ddbClient, circuitTable)
	if err != nil {
		t.Fatalf("failed to create new test table: %v", err)
	}
	defer func() {
		ddbClient.DeleteTable(context.TODO(), &dynamodb.DeleteTableInput{
			TableName: aws.String(circuitTable),
		})
	}()

	err = backend.Store(context.TODO(), key, expected)
	if err != nil {
		t.Fatalf("couldn't store circuit information, got err = %v", err)
	}

	actual, err := backend.Retrieve(context.TODO(), key)
	if err != nil {
		t.Fatalf("expected to get stored CircuitInformation, got err = %v", err)
	}

	if expected != actual {
		t.Fatalf("expected to get %+v; got %+v", expected, actual)
	}
}

func TestBackendIntegrationLockNoTable(t *testing.T) {
	maybeSkip(t)
	t.Parallel()

	ddbClient := dynamodbClientFromURL()
	testID := uuid.NewString()
	key := fmt.Sprintf("circuit-breaker-%s", testID)

	circuitTable := fmt.Sprintf("circuit_info_%s", testID)
	locksTable := fmt.Sprintf("circuit_breaker_locks_%s", testID)
	lockClient, err := ddblock.New(ddbClient, locksTable)
	if err != nil {
		t.Fatalf("could not create dynamodb lock client: %v", err)
	}
	backend := ddbbackend.Backend{
		Client:           ddbClient,
		LockClient:       lockClient,
		CircuitTableName: circuitTable,
		LockTableName:    locksTable,
	}

	_, err = backend.Lock(context.TODO(), key)
	var notFound *types.ResourceNotFoundException
	if !errors.As(err, &notFound) {
		t.Fatalf("expected err to Unwrap to ResourceNotFoundException; got err = %v", err)
	}
}

func TestBackendIntegrationLockWithTable(t *testing.T) {
	maybeSkip(t)
	t.Parallel()

	ddbClient := dynamodbClientFromURL()
	testID := uuid.NewString()
	key := fmt.Sprintf("circuit-breaker-%s", testID)

	circuitTable := fmt.Sprintf("circuit_info_%s", testID)
	locksTable := fmt.Sprintf("circuit_breaker_locks_%s", testID)
	lockClient, err := ddblock.New(ddbClient, locksTable)
	if err != nil {
		t.Fatalf("could not create dynamodb lock client: %v", err)
	}
	table, err := lockClient.CreateTable(locksTable)
	if err != nil {
		t.Fatalf("expected to be able to create a locks table, got err = %v", err)
	}
	if table.TableDescription.TableStatus != types.TableStatusActive {
		t.Fatalf("cannot test locking with inactive locks table, status = %s", table.TableDescription.TableStatus)
	}
	backend := ddbbackend.Backend{
		Client:           ddbClient,
		LockClient:       lockClient,
		CircuitTableName: circuitTable,
		LockTableName:    locksTable,
	}

	lock, err := backend.Lock(context.TODO(), key)
	if err != nil {
		t.Fatalf("expected to create a lock, got err = %v", err)
	}
	if lock == nil {
		t.Fatalf("expected to get a lock but got nil")
	}
	rawLock, err := lockClient.Get(key)
	if err != nil {
		t.Fatalf("could not get raw lock from Dynamo, got err = %v", err)
	}
	if rawLock.IsExpired() {
		t.Fatal("lock is already expired")
	}
	lock.Unlock()
}
