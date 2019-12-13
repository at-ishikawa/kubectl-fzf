build:
	mkdir -p ./bin
	go build -o ./bin/ ./cmd/kubectl-fzf

test:
	go test -v ./...

test-cover:
	go test -cover -v ./...

install:
	go install ./cmd/kubectl-fzf
