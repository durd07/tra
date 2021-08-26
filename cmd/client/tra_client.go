package main

import (
	"fmt"
	"context"
	"log"
	"io"
	"time"
	"bytes"
	"io/ioutil"
	"encoding/json"
	"net/http"

	pb "github.com/durd07/tra/proto"
	"google.golang.org/grpc"
)

const (
	address = "127.0.0.1:50053"
)

func recvNotification(stream pb.TraService_SubscribeClient) {
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			log.Fatalf("EOF error ", err)
			break
		}

		if err != nil {
			log.Fatalf("failed to recv %v", err)
		}

		log.Println(resp, err)
	}
}

func main() {
	lskpmcs := map[string]string{}
	lskpmcs["I3F2"] = "192.169.60.229"
	lskpmcs["S3F2"] = "192.169.60.228"

	if bs, err := json.Marshal(lskpmcs); err == nil {
		req := bytes.NewBuffer([]byte(bs))

		body_type := "application/json;charset=utf-8"
		resp, _ := http.Post("http://127.0.0.1:50052/lskpmcs", body_type, req)
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))

		resp, _ = http.Get("http://127.0.0.1:50052/lskpmcs")
		body, _ = ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
	} else {
		fmt.Println(err)
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewTraServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*100000)
	defer cancel()

	stream, err := c.Subscribe(ctx, &pb.TraRequest{})
	if err != nil {
		log.Fatalf("could not query node : %v", err)
	}

	recvNotification(stream)
	stream.CloseSend()
}
