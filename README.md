mkdir -p $GO_PATH/src/github.com/durd07/tra/proto
protoc --go_out=$GO_PATH/src/github.com/durd07/tra/proto --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/tra.proto


edit test
