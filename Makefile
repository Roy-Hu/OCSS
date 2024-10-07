# Variables
BINARY_NAME=ocss
SOURCE_DIR=cmd
BIN_DIR=bin

# Default target
.PHONY: all
all: build

# Build target
.PHONY: build
build:
	mkdir -p $(BIN_DIR)
	cd $(SOURCE_DIR) && go build -o ../$(BIN_DIR)/$(BINARY_NAME) main.go

# Clean target
.PHONY: clean
clean:
	rm -f $(BIN_DIR)/$(BINARY_NAME)

# Run target
.PHONY: run
run: build
	./$(BIN_DIR)/$(BINARY_NAME)
