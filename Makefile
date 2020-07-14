GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BUILD_DIR=_docker
DOCKER_NAME=docs-prox
TAG=1.0.0-snapshot
BINARY_NAME=$(BUILD_DIR)/main
MAIN_FILE=cmd/main.go
OUT_DIR=_out
COVER_FILE=$(OUT_DIR)/test.cover
COVER_PKG=$(shell go list ./... | grep -v "**test\|cmd" | paste -s -d"," -)
BENCH_N=0
BENCH_TIME=10s
BENCH_MEM_FILE=$(OUT_DIR)/memprofile_$(BENCH_N).out
BENCH_CPU_FILE=$(OUT_DIR)/cpuprofile_$(BENCH_N).out
BENCH_DIR=decoder

all: clean verify build docker
run: docker
	docker run $(DOCKER_NAME):$(TAG)
docker: build build-ui
	docker build $(BUILD_DIR) -t $(DOCKER_NAME):$(TAG)
build: deps
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux GO111MODULE=on $(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_FILE)
build-ui:
	rm -rf _docs-prox-ui/build
	cd _docs-prox-ui && yarn build
	rm -rf $(BUILD_DIR)/dist
	cp -r _docs-prox-ui/build $(BUILD_DIR)/dist
verify: race bench
bench: outdir
	$(GOTEST) ./$(BENCH_DIR) -bench=. -benchtime $(BENCH_TIME) -benchmem -memprofile "$(BENCH_MEM_FILE)" -cpuprofile "$(BENCH_CPU_FILE)"
cover: test
	go tool cover -html $(COVER_FILE)
race: deps lint outdir
	$(GOTEST) -coverpkg $(COVER_PKG) -coverprofile "$(COVERFILE)" -race ./... -v
	go tool cover -func $(COVER_FILE) | grep total
test: deps lint outdir
	$(GOTEST) -coverpkg $(COVER_PKG) -coverprofile "$(COVER_FILE)" ./... -v
	go tool cover -func $(COVER_FILE) | grep total
lint:
	go mod tidy
	go fmt ./...
	go vet ./...
	golint ./...
outdir:
	mkdir -p $(OUT_DIR)
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BENCH_DIR).test
deps:
	$(GOCMD) mod download
