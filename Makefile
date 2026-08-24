.PHONY: all build-edge build-cloud build-arm run-edge run-cloud clean

all: build-edge build-cloud

# Build edge agent for local Linux
build-edge:
	@echo "🔨 Building Go Edge Agent for Linux..."
	go build -o bin/solaria-edge ./cmd/edge-agent

# Cross-compile edge agent for Raspberry Pi (ARM64)
build-arm:
	@echo "🔨 Cross-compiling Go Edge Agent for Raspberry Pi (linux/arm64)..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/solaria-edge-arm64 ./cmd/edge-agent

# Build cloud server
build-cloud:
	@echo "🔨 Building Go Cloud Run Server..."
	go build -o bin/solaria-cloud ./cmd/cloud-server

# Run edge agent locally with config
run-edge: build-edge
	./bin/solaria-edge --config edge/config.yaml

# Run cloud server locally on port 8080
run-cloud: build-cloud
	PORT=8080 ./bin/solaria-cloud

clean:
	rm -rf bin/
