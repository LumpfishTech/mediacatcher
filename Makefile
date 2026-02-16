.PHONY: build run clean install test

# Build the application
build:
	mkdir -p LumpfishMediaCatcher
	go build -o LumpfishMediaCatcher/LumpfishMediaCatcher ./cmd/lumpfishmediacatcher

# Run the application
run: build
	./LumpfishMediaCatcher/LumpfishMediaCatcher

# Clean build artifacts
clean:
	rm -rf LumpfishMediaCatcher

# Install dependencies
install:
	go mod download

# Run tests
test:
	go test -v ./...

