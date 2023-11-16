cover: COVERAGE_FILE := coverage.out
cover:
	@go test ./... -coverprofile=$(COVERAGE_FILE)  -gcflags=all=-l && \
	go tool cover -html=$(COVERAGE_FILE) && rm $(COVERAGE_FILE)
