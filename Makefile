.PHONY: build install test clean

VERSION ?= dev
BINARY_NAME = ai-reviewer
BUILD_DIR = bin

# Build variables
GO_BUILD_FLAGS ?= -v

# Determine OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	OS = osx
else
	OS = linux
endif

# Set the binary path based on OS
ifeq ($(OS),osx)
	BINARY_PATH = $(BUILD_DIR)/$(BINARY_NAME)
else
	BINARY_PATH = $(BUILD_DIR)/$(BINARY_NAME)
endif

# Default target
all: build

# Build the plugin
build:
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -o $(BINARY_PATH) .
	@echo "Built $(BINARY_PATH)"

# Install the plugin to Bitrise
install: build
	bitrise plugin install --source .

# Run tests
test:
	go test ./...

# Clean up
clean:
	rm -rf $(BUILD_DIR)

# Release with version
release:
	@echo "Creating release for version $(VERSION)"
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-X 'github.com/birmacher/bitrise-plugins-ai-reviewer/version.Version=$(VERSION)'" -o $(BINARY_PATH) .
	@echo "Built release $(BINARY_PATH) with version $(VERSION)"
