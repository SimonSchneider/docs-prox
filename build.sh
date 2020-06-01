export GO111MODULE=on
export CGO_ENABLED=0
export GOARCH=amd64
export GOOS=linux

go mod download
go build -o docs-prox ./cmd/*.go

docker build . -t docs-prox:$1
