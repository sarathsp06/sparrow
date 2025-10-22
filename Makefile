build:
	go build -o grpc-server ./cmd/grpc-server

build-ko:
	ko build -o grpc-server ./cmd/grpc-server

example:
	DATABASE_URL='postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable' go run examples/grpc_client.go 

run: build
	DATABASE_URL='postgres://riveruser:riverpass@0.0.0.0:5432/riverqueue?sslmode=disable' ./grpc-server

test:
	go test -v

clean:
	rm -f grpc-server

proto:
	buf generate

docker-dev:
	docker-compose -f docker-compose.dev.yml up -d
	
.PHONY: build run test clean proto docker-dev example