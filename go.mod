module github.com/sigmavirus24/circuitry

go 1.21

require (
	github.com/bsm/redislock v0.9.4
	github.com/go-redis/redismock/v9 v9.2.0
	github.com/redis/go-redis/v9 v9.5.3 // Until https://github.com/go-redis/redismock/pull/85/files is merged
	github.com/sirupsen/logrus v1.9.3
)

require (
	cirello.io/dynamolock/v2 v2.0.3
	github.com/aws/aws-sdk-go-v2 v1.30.1
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.14.7
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.7.29
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.34.1
	github.com/aws/smithy-go v1.20.3
	github.com/google/uuid v1.6.0
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.22.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.9.14 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
)
