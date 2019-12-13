build:
	mkdir -p ./bin
	go build -o ./bin/ ./cmd/kubectl-fzf

install:
	go install ./cmd/kubectl-fzf
