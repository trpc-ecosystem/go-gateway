GO = go
DTOOLS = dtools
TARGET = trpc-gateway

# set for mac M1Pro
export GOARCH=amd64

all: fmt goimports lint

fmt:
	@gofmt -s -w ./$*

vet:
	@go vet -all ./

ec:
	@errcheck ./...  | grep -v Close

lint:
	@golint ./...

goimports:
	@goimports -d -w ./

test:
	CFLAGS=-g
	export CFLAGS
	$(GO) test $(M)  -v -gcflags=all=-l -coverpkg=./... -coverprofile=test.out ./...
clean:
	rm -f $(TARGET)
	rm -rf release

cover: COVERAGE_FILE := coverage.out
cover:
	@go test ./... -coverprofile=$(COVERAGE_FILE)  -gcflags=all=-l && \
	go tool cover -html=$(COVERAGE_FILE) && rm $(COVERAGE_FILE)

mod:
	cd plugin/accesslog && go mod tidy
	cd plugin/batchrequest && go mod tidy
	cd plugin/cors && go mod tidy
	cd plugin/devenv && go mod tidy
	cd plugin/limiter/polaris && go mod tidy
	cd plugin/logreplay && go mod tidy
	cd plugin/mocking && go mod tidy
	cd plugin/polaris/canaryrouter && go mod tidy
	cd plugin/polaris/metarouter && go mod tidy
	cd plugin/polaris/setrouter && go mod tidy
	cd plugin/redirect && go mod tidy
	cd plugin/routercheck && go mod tidy
	cd plugin/traceid && go mod tidy
	cd plugin/transformer/request && go mod tidy
	cd plugin/transformer/response && go mod tidy
	cd plugin/transformer/trpcerr2body && go mod tidy
	cd core/loader/etcd && go mod tidy
	cd core/service/protocol/grpc && go mod tidy
	cd example/loader/etcd && go mod tidy
	cd example/loader/file && go mod tidy