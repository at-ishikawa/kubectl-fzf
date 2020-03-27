goimports:
	goimports -w .

lint:
	golangci-lint run -v ./...

build:
	mkdir -p ./bin
	go build -o ./bin/ ./cmd/kubectl-fzf

generate:
	go generate -v ./...

test:
	go test -v ./...

test-cover:
	go test -cover -v ./...

install:
	go install ./cmd/kubectl-fzf
