module github.com/sigmavirus24/circuitry

go 1.23.0

toolchain go1.23.7

require (
	github.com/bsm/redislock v0.9.4
	github.com/go-redis/redismock/v9 v9.2.0
	github.com/redis/go-redis/v9 v9.12.1 // Until https://github.com/go-redis/redismock/pull/85/files is merged
	github.com/sirupsen/logrus v1.9.3
)

require (
	cirello.io/dynamolock/v2 v2.1.0
	github.com/aws/aws-sdk-go-v2 v1.38.1
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.20.4
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.8.4
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.49.1
	github.com/aws/smithy-go v1.22.5
	github.com/google/uuid v1.6.0
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.29.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.4 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
)
