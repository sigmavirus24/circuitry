module github.com/sigmavirus24/circuitry

go 1.24

require (
	github.com/bsm/redislock v0.9.4
	github.com/go-redis/redismock/v9 v9.2.0
	github.com/redis/go-redis/v9 v9.18.0 // Until https://github.com/go-redis/redismock/pull/85/files is merged
	github.com/sirupsen/logrus v1.9.4
)

require (
	cirello.io/dynamolock/v2 v2.1.0
	github.com/aws/aws-sdk-go-v2 v1.41.4
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.20.35
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.8.35
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.56.2
	github.com/aws/smithy-go v1.24.2
	github.com/google/uuid v1.6.0
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.32.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.20 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/telemetry v0.0.0-20240522233618-39ace7a40ae7 // indirect
	golang.org/x/tools v0.29.0 // indirect
	golang.org/x/vuln v1.1.4 // indirect
)

tool golang.org/x/vuln/cmd/govulncheck
