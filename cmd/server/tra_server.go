package main

import (
	"context"
	"log"
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
func (s *server) UpdateLskpmc(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)
	key := in.GetUpdateLskpmcRequest().GetLskpmc().Key
	val := in.GetUpdateLskpmcRequest().GetLskpmc().Val
	log.Printf("GRPC UpdateLskpmc Received from %s : %v=%v", p.Addr.String(), key, val)

	lskpmcs[key] = val

	return &pb.TraServiceResponse{ Response: &pb.TraServiceResponse_UpdateLskpmcResponse{UpdateLskpmcResponse : &pb.UpdateLskpmcResponse{Ret: 0} }}, nil
}

func (s *server) GetIpFromLskpmc(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)
	log.Printf("GRPC GetIpFromLskpmc Received from %s : %v", p.Addr.String(), in)
	key := in.GetGetIpFromLskpmcRequest().GetKey()
	return &pb.TraServiceResponse{ Response: &pb.TraServiceResponse_GetIpFromLskpmcResponse{GetIpFromLskpmcResponse: &pb.GetIpFromLskpmcResponse{Lskpmc: &pb.Lskpmc{Key: key, Val: lskpmcs[key]}}}}, nil
}

func (s *server) Subscribe(in *pb.TraServiceRequest, stream pb.TraService_SubscribeServer) error {
	p, _ := peer.FromContext(stream.Context())
	peer_addr := p.Addr.String()
	log.Printf("GRPC Subscribe Recieved from %s", peer_addr)

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
		log.Printf("GRPC Notify %+v", streams)
		for stream, _ := range streams {
			if err := stream.Context().Err(); err != nil {
				delete(streams, stream)
				continue
			}

			r := pb.SubscribeResponse{Lskpmcs: []*pb.Lskpmc{}}

			for k, v := range lskpmcs {
				lskpmc := pb.Lskpmc{Key :k, Val: v}
				r.Lskpmcs = append(r.Lskpmcs, &lskpmc)
			}
			resp := pb.TraServiceResponse{ Response: &pb.TraServiceResponse_SubscribeResponse{SubscribeResponse: &r}}
			stream.Send(&resp)

			p, _ := peer.FromContext(stream.Context())
			log.Printf("     send to %+v", p.Addr.String())
		}
	}
}

func grpcServer() {
	log.Printf("Starting GRPC Server %s", port)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Printf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	myserver := server{}
	go myserver.Notify()

	pb.RegisterTraServiceServer(s, &myserver)
	if err := s.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
	}
}


func lskpmcsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Read failed:", err)
		}
		defer r.Body.Close()

		err = json.Unmarshal(b, &lskpmcs)
		if err != nil {
			log.Printf("json format error:", err)
		}
		log.Printf("%s update_data: %#v", r.Method, lskpmcs)

		change_chan <- struct{}{}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	} else if r.Method == "GET" {
		b, err := json.Marshal(lskpmcs)
		if err != nil {
			log.Printf("json format error:", err)
			return
		}

		log.Printf("%s get_data: %#v", r.Method, lskpmcs)

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
	log.Printf("Starting HTTP Server %s", ":50052")
	http.HandleFunc("/lskpmcs", lskpmcsHandler)
	http.ListenAndServe(":50052", nil)
}


func main() {
	go grpcServer()
	go httpServer()

	// Wait forever
	select {}
}
