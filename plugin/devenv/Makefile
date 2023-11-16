# Targets:
#
#   all:          Builds the code locally after testing
#
#   fmt:          Formats the source files
#   build_dev:    Builds the code locally
#   build         Build production binary file.
#   vet:          Vets the code
#   lint:         Runs lint over the code (you do not need to fix everything)
#   test:         Runs the tests
#   cover:        Gives you the URL to a nice test coverage report


export GO111MODULE=on
export GOPRIVATE=git.code.oa.com
export GOPROXY=goproxy.cn
export GOSUMDB="sum.woa.com+643d7a06+Ac5f5VOC4N8NUXdmhbm8pZSXIWfhek5JSmWdWrq7pLX4"

# set for M1Pro
export GOARCH=amd64

# The first target is always the default action if `make` is called without
# args we build and install into $GOPATH so that it can just be run

all: fmt goimports vet lint

fmt:
	@gofmt -s -w ./$*
gen:
	@go generate

vet:
	@go vet -all ./

ec:
	@errcheck ./...  | grep -v Close

lint:
	@golint ./...

goimports:
	@goimports -d -w ./


cover: COVERAGE_FILE := coverage.out
cover:
	@go test ./... -coverprofile=$(COVERAGE_FILE)  -gcflags=all=-l && \
	go tool cover -html=$(COVERAGE_FILE) && rm $(COVERAGE_FILE)



