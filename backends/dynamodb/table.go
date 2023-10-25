package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	ddbexp "github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	ddb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/sigmavirus24/circuitry"
)

// CreateCircuitInformationTableOptions has the options supported for creating
// the CircuitInformation table
type CreateCircuitInformationTableOptions struct {
	BillingMode           ddbtypes.BillingMode
	ProvisionedThroughput *ddbtypes.ProvisionedThroughput
	Tags                  []ddbtypes.Tag
}

// CreateCircuitInformationTableOption configures the
// CreateCircuitInformationTable function behaviour
type CreateCircuitInformationTableOption func(*CreateCircuitInformationTableOptions)

// CreateTableWithTags adds tags to the table creation call
func CreateTableWithTags(tags []ddbtypes.Tag) CreateCircuitInformationTableOption {
	return func(o *CreateCircuitInformationTableOptions) {
		o.Tags = append(o.Tags, tags...)
	}
}

// CreateTableWithProvisionedThroughput will configure the ProvisionedThroughput for the
// new DynamoDB table
func CreateTableWithProvisionedThroughput(throughput *ddbtypes.ProvisionedThroughput) CreateCircuitInformationTableOption {
	return func(o *CreateCircuitInformationTableOptions) {
		o.ProvisionedThroughput = throughput
	}
}

// CreateTableWithBillingMode will configure the BillingMode for the new DynamoDB table
func CreateTableWithBillingMode(mode ddbtypes.BillingMode) CreateCircuitInformationTableOption {
	return func(o *CreateCircuitInformationTableOptions) {
		o.BillingMode = mode
	}
}

// KeyName is the name of the DynamoDB Hash Key Attribute
const KeyName = "breaker_name"

// CreateCircuitInformationTable is a helper for creating a table to store
// Circuit Breaker Information (a.k.a., circuitry.CircuitInformation) in
// DynamoDB
func CreateCircuitInformationTable(ctx context.Context, client DynamoClient, tableName string, opts ...CreateCircuitInformationTableOption) (*ddb.CreateTableOutput, error) {
	options := CreateCircuitInformationTableOptions{BillingMode: ddbtypes.BillingModePayPerRequest}
	for _, opt := range opts {
		opt(&options)
	}
	keySchema := []ddbtypes.KeySchemaElement{
		{
			AttributeName: aws.String(KeyName),
			KeyType:       ddbtypes.KeyTypeHash,
		},
	}
	attributes := []ddbtypes.AttributeDefinition{
		{
			AttributeName: aws.String(KeyName),
			AttributeType: ddbtypes.ScalarAttributeTypeS,
		},
	}
	input := &ddb.CreateTableInput{
		AttributeDefinitions: attributes,
		KeySchema:            keySchema,
		TableName:            aws.String(tableName),
		BillingMode:          options.BillingMode,
	}

	if t := options.ProvisionedThroughput; t != nil {
		input.ProvisionedThroughput = t
	}

	if ts := options.Tags; ts != nil {
		input.Tags = ts
	}
	output, err := client.CreateTable(ctx, input)
	if err != nil {
		return nil, &RemoteBackendError{
			Err:       err,
			Operation: OpCreateTable,
			TableName: tableName,
		}
	}
	return output, nil
}

type circuitInfoRecord struct {
	Name                 string    `dynamodbav:"breaker_name"`
	State                uint64    `dynamodbav:"state"`
	Generation           uint64    `dynamodbav:"generation"`
	ConsecutiveFailures  uint64    `dynamodbav:"consecutive_failures"`
	ConsecutiveSuccesses uint64    `dynamodbav:"consecutive_successes"`
	Total                uint64    `dynamodbav:"total"`
	TotalFailures        uint64    `dynamodbav:"total_failures"`
	TotalSuccesses       uint64    `dynamodbav:"total_successes"`
	ExpiresAfter         time.Time `dynamodbav:"expires_after"`
}

func (r circuitInfoRecord) ToCircuitInformation() circuitry.CircuitInformation {
	return circuitry.CircuitInformation{
		State:                circuitry.CircuitState(r.State),
		Generation:           r.Generation,
		ConsecutiveFailures:  r.ConsecutiveFailures,
		ConsecutiveSuccesses: r.ConsecutiveSuccesses,
		Total:                r.Total,
		TotalFailures:        r.TotalFailures,
		TotalSuccesses:       r.TotalSuccesses,
		ExpiresAfter:         r.ExpiresAfter,
	}
}

func (r circuitInfoRecord) ToUpdateExpression() (*ddbexp.Expression, error) {
	update := ddbexp.Set(ddbexp.Name("state"), ddbexp.Value(r.State)).
		Set(ddbexp.Name("generation"), ddbexp.Value(r.Generation)).
		Set(ddbexp.Name("consecutive_failures"), ddbexp.Value(r.ConsecutiveFailures)).
		Set(ddbexp.Name("consecutive_successes"), ddbexp.Value(r.ConsecutiveSuccesses)).
		Set(ddbexp.Name("expires_after"), ddbexp.Value(r.ExpiresAfter)).
		Set(ddbexp.Name("total"), ddbexp.Value(r.Total)).
		Set(ddbexp.Name("total_failures"), ddbexp.Value(r.TotalFailures)).
		Set(ddbexp.Name("total_successes"), ddbexp.Value(r.TotalSuccesses))
	exp, err := ddbexp.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return nil, err
	}
	return &exp, nil
}

func recordFromCircuitInformation(ci circuitry.CircuitInformation) circuitInfoRecord {
	return circuitInfoRecord{
		Generation:           ci.Generation,
		ConsecutiveFailures:  ci.ConsecutiveFailures,
		ConsecutiveSuccesses: ci.ConsecutiveSuccesses,
		Total:                ci.Total,
		TotalFailures:        ci.TotalFailures,
		TotalSuccesses:       ci.TotalSuccesses,
		ExpiresAfter:         ci.ExpiresAfter,
	}
}
