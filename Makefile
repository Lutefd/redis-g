run: build
	@./bin/redis-g --listenAddr :5001

build:
	@go build -o bin/redis-g .
