package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"net/http"
	"io/ioutil"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	pb "github.com/durd07/tra/proto"
)

var (
	lskpmcs = map[string]string{}
)

const (
	port = ":50053"
)

var (
	streams = make(map[pb.TraService_SubscribeServer]struct{})
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedTraServiceServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) UpdateLskpmc(ctx context.Context, in *pb.Lskpmc) (*pb.String, error) {
	p, _ := peer.FromContext(ctx)
	fmt.Printf("GRPC UpdateLskpmc Received from %s : %v=%v", p.Addr.String(), in.Key, in.Val)

	lskpmcs[in.Key] = in.Val

	return &pb.String{}, nil
}

func (s *server) GetIpFromLskpmc(ctx context.Context, in *pb.String) (*pb.String, error) {
	p, _ := peer.FromContext(ctx)
	fmt.Printf("GRPC GetIpFromLskpmc Received from %s : %v", p.Addr.String(), in)
	return &pb.String{Data: lskpmcs[in.Data]}, nil
}

func (s *server) Subscribe(in *pb.String, stream pb.TraService_SubscribeServer) error {
	p, _ := peer.FromContext(stream.Context())
	peer_addr := p.Addr.String()
	fmt.Printf("GRPC Subscribe Recieved from %s", peer_addr)

	streams[stream] = struct{}{}

	change_chan <- struct{}{}
	for {
		if err := stream.Context().Err(); err != nil {
			delete(streams, stream)
			break
		}
		time.Sleep(time.Second)
	}
	return nil
}

var change_chan = make(chan struct{})

func (s *server) Notify() error {
	for {
		_ = <-change_chan
		fmt.Printf("GRPC Notify %+v\n", streams)
		for stream, _ := range streams {
			if err := stream.Context().Err(); err != nil {
				delete(streams, stream)
				continue
			}

			resp := pb.LskpmcResponse{Lskpmcs: []*pb.Lskpmc{}}
			for k, v := range lskpmcs {
				lskpmc := pb.Lskpmc{Key :k, Val: v}
				resp.Lskpmcs = append(resp.Lskpmcs, &lskpmc)
			}
			stream.Send(&resp)

			p, _ := peer.FromContext(stream.Context())
			fmt.Printf("     send to %+v\n", p.Addr.String())
		}
	}
}

func grpcServer() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	myserver := server{}
	go myserver.Notify()

	pb.RegisterTraServiceServer(s, &myserver)
	if err := s.Serve(lis); err != nil {
		fmt.Printf("failed to serve: %v", err)
	}
}


func lskpmcsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("Read failed:", err)
		}
		defer r.Body.Close()

		err = json.Unmarshal(b, &lskpmcs)
		if err != nil {
			fmt.Printf("json format error:", err)
		}
		fmt.Printf("%s update_data: %#v", r.Method, lskpmcs)

		change_chan <- struct{}{}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	} else if r.Method == "GET" {
		b, err := json.Marshal(lskpmcs)
		if err != nil {
			fmt.Printf("json format error:", err)
			return
		}

		fmt.Printf("%s get_data: %#v", r.Method, lskpmcs)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Write(b)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	}
}

func httpServer() {
	http.HandleFunc("/lskpmcs", lskpmcsHandler)
	http.ListenAndServe(":50052", nil)
}


func main() {
	go grpcServer()
	go httpServer()

	// Wait forever
	select {}
}
