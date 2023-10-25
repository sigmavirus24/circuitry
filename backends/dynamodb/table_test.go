package dynamodb_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/sigmavirus24/circuitry/backends/dynamodb"
)

func TestCreateCircuitInformationTable(t *testing.T) {
	testCases := map[string]struct {
		client      dynamodb.DynamoClient
		tableName   string
		opts        []dynamodb.CreateCircuitInformationTableOption
		expectedErr error
	}{
		"no options": {
			newDDBMock(),
			"create-ci-table-000",
			[]dynamodb.CreateCircuitInformationTableOption{},
			nil,
		},
		"with tags": {
			newDDBMock(),
			"create-ci-table-001",
			[]dynamodb.CreateCircuitInformationTableOption{
				dynamodb.CreateTableWithTags([]ddbtypes.Tag{{Key: aws.String("key"), Value: aws.String("tag-val")}}),
			},
			nil,
		},
		"with throughput": {
			newDDBMock(),
			"create-ci-table-002",
			[]dynamodb.CreateCircuitInformationTableOption{
				dynamodb.CreateTableWithProvisionedThroughput(&ddbtypes.ProvisionedThroughput{}),
			},
			nil,
		},
		"with billing mode": {
			newDDBMock(),
			"create-ci-table-003",
			[]dynamodb.CreateCircuitInformationTableOption{
				dynamodb.CreateTableWithBillingMode(ddbtypes.BillingModePayPerRequest),
			},
			nil,
		},
		"with all options": {
			newDDBMock(),
			"create-ci-table-004",
			[]dynamodb.CreateCircuitInformationTableOption{
				dynamodb.CreateTableWithTags([]ddbtypes.Tag{
					{Key: aws.String("key001"), Value: aws.String("tag-val-001")},
					{Key: aws.String("key002"), Value: aws.String("tag-val-002")},
				}),
				dynamodb.CreateTableWithProvisionedThroughput(&ddbtypes.ProvisionedThroughput{}),
				dynamodb.CreateTableWithBillingMode(ddbtypes.BillingModePayPerRequest),
			},
			nil,
		},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := dynamodb.CreateCircuitInformationTable(context.TODO(), tc.client, tc.tableName, tc.opts...)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected CreateCircuitInformationTable to return err = %v; got %v", tc.expectedErr, err)
			}
		})
	}
}
