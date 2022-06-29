ROOT_DIR := $(shell pwd)
SRC_DIR := $(ROOT_DIR)/src
PROTO_DIR := $(ROOT_DIR)/proto
PROTO_FILE := $(PROTO_DIR)/notion_backup.proto
PROTOC := protoc
BINARY := notion_backup

.PHONY: test
test:
	go test -v $(SRC_DIR)/...

.PHONY: install
install:
	go install .
	
.PHONY: build
build:
	go build -o $(BINARY)

.PHONY: proto
proto:
	$(PROTOC) -I=$(PROTO_DIR) --go_out=$(ROOT_DIR) $(PROTO_FILE)

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: fmt
fmt:
	go fmt $(SRC_DIR)/...

.PHONY: all
all: fmt tidy test

.PHONY: clean
clean:
	go clean