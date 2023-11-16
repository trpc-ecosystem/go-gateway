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
