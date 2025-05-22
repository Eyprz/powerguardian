# Makefile for Go cross-compilation

# --- Variables ---
# The name of the binary to be built
BINARY_NAME := pg-agent
# The name of go package
PACKAGE_NAME := powerguardian

# Target Operating System (e.g., linux, windows, darwin)
TARGET_OS := linux
# Target Architecture (e.g., amd64, arm, 386)
TARGET_ARCH := arm
# Target ARM version (if TARGET_ARCH is arm, e.g., 5, 6, 7)
TARGET_ARM_VERSION := 6

# Go command
GO := go

# Build flags (e.g., -ldflags="-s -w")
BUILD_FLAGS :=

# --- Targets ---

# Default target: build the binary
.PHONY: all
all: build

# Build the binary for the specified target
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for $(TARGET_OS)/$(TARGET_ARCH) (ARMv$(TARGET_ARM_VERSION))..."
	@GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH) GOARM=$(TARGET_ARM_VERSION) $(GO) build $(BUILD_FLAGS) -o $(BINARY_NAME) ./cmd/$(PACKAGE_NAME)
	@echo "$(BINARY_NAME) built successfully for $(TARGET_OS)/$(TARGET_ARCH) (ARMv$(TARGET_ARM_VERSION))."

# Clean up build artifacts
.PHONY: clean
clean:
	@echo "Cleaning up build artifacts..."
	@if [ -f "$(BINARY_NAME)" ]; then \
		rm -f $(BINARY_NAME); \
		echo "Removed $(BINARY_NAME)"; \
	else \
		echo "$(BINARY_NAME) not found, nothing to clean."; \
	fi
	@echo "Cleanup complete."

# Help target to display available commands
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build    Build the application for the target platform."
	@echo "  make clean    Remove build artifacts."
	@echo "  make help     Show this help message."
	@echo ""
	@echo "Configuration:"
	@echo "  BINARY_NAME      : $(BINARY_NAME)"
	@echo "  TARGET_OS        : $(TARGET_OS)"
	@echo "  TARGET_ARCH      : $(TARGET_ARCH)"
	@echo "  TARGET_ARM_VERSION : $(TARGET_ARM_VERSION)"
