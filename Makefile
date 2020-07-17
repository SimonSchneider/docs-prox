GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BUILD_DIR=docker
DOCKER_NAME=docs-prox
TAG=1.0.0-snapshot
BINARY_NAME=$(BUILD_DIR)/main
MAIN_FILE=pkg/cmd/main.go
OUT_DIR=out
COVER_FILE=$(OUT_DIR)/test.cover
COVER_PKG=$(shell go list $(ROOT_GO_DIR) | grep -v "**test\|cmd" | paste -s -d"," -)
BENCH_N=0
BENCH_TIME=10s
BENCH_MEM_FILE=$(OUT_DIR)/memprofile_$(BENCH_N).out
BENCH_CPU_FILE=$(OUT_DIR)/cpuprofile_$(BENCH_N).out
ROOT_GO_DIR=./pkg/...

all: clean verify docker
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
verify: clean race
cover: test
	go tool cover -html $(COVER_FILE)
race: deps lint outdir
	$(GOTEST) -coverpkg $(COVER_PKG) -coverprofile "$(COVERFILE)" -race $(ROOT_GO_DIR) -v
	go tool cover -func $(COVER_FILE) | grep total
test: deps lint outdir
	$(GOTEST) -coverpkg $(COVER_PKG) -coverprofile "$(COVER_FILE)" $(ROOT_GO_DIR) -v
	go tool cover -func $(COVER_FILE) | grep total
lint:
	go mod tidy
	go fmt $(ROOT_GO_DIR)
	go vet $(ROOT_GO_DIR)
	golint $(ROOT_GO_DIR)
outdir:
	mkdir -p $(OUT_DIR)
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BENCH_DIR).test
deps:
	$(GOCMD) mod download
