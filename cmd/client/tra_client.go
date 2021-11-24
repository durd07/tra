package main

import (
	"context"
	"log"
	"io"
	//"time"
	"bytes"
	"io/ioutil"
	"encoding/json"
	"net/http"

	pb "github.com/durd07/tra/proto"
	"google.golang.org/grpc"
)

const (
	//grpc_addr = "istio-tra.cncs.svc.cluster.local:50053"
	//http_addr = "istio-tra.cncs.svc.cluster.local:50053"
	grpc_addr = "127.0.0.1:50053"
	http_addr = "127.0.0.1:50052"
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

		log.Printf("RECV NOTIFY %v %v\n", resp.GetSubscribeResponse().GetData(), err)
	}
}

func main() {
	lskpmcs := map[string]string{}
	lskpmcs["I3F2"] = "192.169.60.229"
	lskpmcs["S3F2"] = "192.169.60.228"

	if bs, err := json.Marshal(lskpmcs); err == nil {
		req := bytes.NewBuffer([]byte(bs))

		body_type := "application/json;charset=utf-8"
		resp, err := http.Post("http://" + http_addr + "/lskpmcs", body_type, req)
		if err != nil {
			log.Fatalf("http connection failed %v", err.Error())
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("POST resp %s\n", string(body))

		resp, _ = http.Get("http://" + http_addr + "/lskpmcs")
		body, _ = ioutil.ReadAll(resp.Body)
		log.Printf("GET  resp %s\n", string(body))
	} else {
		log.Println(err)
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial(grpc_addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewTraServiceClient(conn)

	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*100000)
	//defer cancel()
	ctx := context.Background()

	log.Printf("GRPC update lskpmc %s\n", "S1F1=192.168.60.001")

	c.Update(ctx, &pb.TraServiceRequest{Type: "lskpmc", Request: &pb.TraServiceRequest_UpdateRequest{UpdateRequest: &pb.UpdateRequest{Data: map[string]string{"S1F1": "192.168.60.001"}}}})

	resp, err := c.Retrieve(ctx, &pb.TraServiceRequest{Type: "lskpmc", Request: &pb.TraServiceRequest_RetrieveRequest{RetrieveRequest: &pb.RetrieveRequest{Key: "XXXX"}}})
	log.Printf("GRPC retrieve lskpmc XXXX %v", resp)

	stream, err := c.Subscribe(ctx, &pb.TraServiceRequest{Type:"lskpmc"})
	if err != nil {
		log.Fatalf("could not query node : %v", err)
	}

	recvNotification(stream)
	stream.CloseSend()
}
