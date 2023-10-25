.PHONY: all

files := $(wildcard *.go **/*.go)

all: lint test

lint: $(files)
	@echo "Running autoformatters ..."
	@go fmt .
	@goimports -w .
	@echo "Running golangci-lint ..."
	@golangci-lint run . 

test: $(files)
	@go test -v -cover -coverprofile=coverage.out . ./...

integration-test: $(files)
	@DYNAMODB_URL=http://localhost:8000 go test -v -cover -coverprofile=coverage.out ./...

ci-integration-test: $(files)
	@go test -v -cover -coverprofile=coverage.out ./...

coverage.out: test

show-cov: coverage.out
	@go tool cover -func=coverage.out

ci-show-integration-cov: ci-integration-test
	@go tool cover -func=coverage.out

html-cov: coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@python -c 'import pathlib, webbrowser; p = pathlib.Path("./coverage.html").absolute(); webbrowser.open(f"file://{p}")'
