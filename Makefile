.PHONY: lint test integration-test show-cov html-cov

files := $(wildcard *.go)

lint: $(files)
	@echo "Running autoformatters ..."
	@go fmt .
	@goimports -w .
	@echo "Running golangci-lint ..."
	@golangci-lint run . 

test: lint $(files)
	@go test -v -cover -coverprofile=coverage.out . ./...

integration-test: lint
	@DYNAMODB_URL=http://localhost:8000 go test -v -cover -coverprofile=coverage.out ./...

coverage.out: test

show-cov: coverage.out
	@go tool cover -func=coverage.out

show-integration-cov: integration-test
	@go tool cover -func=coverage.out

html-cov: coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@python -c 'import pathlib, webbrowser; p = pathlib.Path("./coverage.html").absolute(); webbrowser.open(f"file://{p}")'
