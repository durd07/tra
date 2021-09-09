GOPATH=$(shell go env GOPATH)

all: docker

generate_protobuf:
	mkdir -p ${GOPATH}/src/github.com/durd07/tra/proto
#	protoc --go_out=${GOPATH}/src/github.com/durd07/tra/proto --go_opt=paths=source_relative \
#		--go-grpc_out=${GOPATH}/src/github.com/durd07/tra/proto --go-grpc_opt=paths=source_relative \
#		proto/tra.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/tra.proto
	cp proto/* ${GOPATH}/src/github.com/durd07/tra/proto

client: generate_protobuf
	go build -o ./cmd/client/tra_client ./cmd/client

server: generate_protobuf
	go build -o ./cmd/server/tra_server ./cmd/server

docker: client server
	cp ./cmd/client/tra_client ./docker
	cp ./cmd/server/tra_server ./docker
	docker build -t felixdu.hz.dynamic.nsn-net.net/tra:v0.1 ./docker
	docker push felixdu.hz.dynamic.nsn-net.net/tra:v0.1
