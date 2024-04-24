run: build
	@./bin/redis-g

build:
	@go build -o bin/redis-g .
