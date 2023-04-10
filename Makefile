# build file
GO_BUILD=go build -ldflags -s -v

BIN_BINARY_NAME=sub_account
sub:
	$(GO_BUILD) -o $(BIN_BINARY_NAME) cmd/main.go
	@echo "Build $(BIN_BINARY_NAME) successfully. You can run ./$(BIN_BINARY_NAME) now.If you can't see it soon,wait some seconds"

SLB_BIN_BINARY_NAME=sub_slb_svr
slb:
	$(GO_BUILD) -o $(SLB_BIN_BINARY_NAME) cmd/lb/main.go
	@echo "Build $(SLB_BIN_BINARY_NAME) successfully. You can run ./$(SLB_BIN_BINARY_NAME) now.If you can't see it soon,wait some seconds"

update:
	go mod tidy -compat=1.17
