mkdir -p $GO_PATH/src/github.com/durd07/tra/proto
protoc --go_out=$GO_PATH/src/github.com/durd07/tra/proto --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/tra.proto


```
apt install -y protobuf-compiler

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

export PATH="$PATH:$(go env GOPATH)/bin"
```
