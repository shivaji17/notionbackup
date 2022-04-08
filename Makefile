ROOT_DIR := $(shell pwd)
SRC_DIR := $(ROOT_DIR)/src

test:
	go test -v $(SRC_DIR)/...

tidy:
	go mod tidy

fmt:
	go fmt $(SRC_DIR)/...

all: fmt tidy test

clean:
	go clean