package main

import (
	"context"
	"log"
	"net"
	"time"

	"encoding/json"
	"io/ioutil"
	"net/http"

	pb "github.com/durd07/tra/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

const (
	http_port = ":50052"
	grpc_port = ":50053"
)

var (
	lskpmcs = make(map[string]string)
	streams = make(map[pb.TraService_SubscribeLskpmcServer]struct{})
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedTraServiceServer
}

func (s *server) CreateLskpmc(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	req_lskpmcs := in.GetCreateLskpmcRequest().GetLskpmcs()
	log.Printf("GRPC Create Received from %s : %v", p.Addr.String(), req_lskpmcs)

	for k, v := range req_lskpmcs {
		lskpmcs[k] = v
	}

	// Once there are update, notify all subscribers
	change_chan <- struct{}{}

	return &pb.TraServiceResponse{Ret: 0, Response: &pb.TraServiceResponse_CreateLskpmcResponse{CreateLskpmcResponse: &pb.CreateLskpmcResponse{}}}, nil
}

func (s *server) UpdateLskpmc(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	req_lskpmcs := in.GetCreateLskpmcRequest().GetLskpmcs()
	log.Printf("GRPC Update Received from %s : %v", p.Addr.String(), req_lskpmcs)

	for k, v := range req_lskpmcs {
		lskpmcs[k] = v
	}

	// Once there are update, notify all subscribers
	change_chan <- struct{}{}

	return &pb.TraServiceResponse{Ret: 0, Response: &pb.TraServiceResponse_UpdateLskpmcResponse{UpdateLskpmcResponse: &pb.UpdateLskpmcResponse{}}}, nil
}

func (s *server) RetrieveLskpmc(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	key := in.GetRetrieveLskpmcRequest().GetLskpmc()
	log.Printf("GRPC Retrieve Received from %s : %v=%v", p.Addr.String(), key, lskpmcs[key])

	return &pb.TraServiceResponse{Ret: 0, Response: &pb.TraServiceResponse_RetrieveLskpmcResponse{RetrieveLskpmcResponse: &pb.RetrieveLskpmcResponse{Lskpmcs: map[string]string{key: lskpmcs[key]}}}}, nil
}

func (s *server) DeleteLskpmc(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	key := in.GetDeleteLskpmcRequest().GetLskpmc()
	log.Printf("GRPC Delete Received from %s : %v", p.Addr.String(), key)

	delete(lskpmcs, key)

	return &pb.TraServiceResponse{Ret: 0, Response: &pb.TraServiceResponse_DeleteLskpmcResponse{DeleteLskpmcResponse: &pb.DeleteLskpmcResponse{}}}, nil
}

func (s *server) SubscribeLskpmc(in *pb.TraServiceRequest, stream pb.TraService_SubscribeLskpmcServer) error {
	p, _ := peer.FromContext(stream.Context())
	peer_addr := p.Addr.String()
	log.Printf("GRPC Subscribe Recieved from %s", peer_addr)

	streams[stream] = struct{}{}

	change_chan <- struct{}{}
	for {
		if err := stream.Context().Err(); err != nil {
			log.Printf("remove %s from stream", p.Addr.String())
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

			resp := pb.TraServiceResponse{Ret: 0, Response: &pb.TraServiceResponse_SubscribeLskpmcResponse{SubscribeLskpmcResponse: &pb.SubscribeLskpmcResponse{Lskpmcs: lskpmcs}}}
			stream.Send(&resp)

			p, _ := peer.FromContext(stream.Context())
			log.Printf("     send to %+v", p.Addr.String())
		}
	}
}

func grpcServer() {
	log.Printf("Starting GRPC Server %s", grpc_port)
	lis, err := net.Listen("tcp", grpc_port)
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

		log.Printf("%s  get_data: %#v", r.Method, lskpmcs)

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
	log.Printf("Starting HTTP Server %s", http_port)
	http.HandleFunc("/lskpmcs", lskpmcsHandler)
	http.ListenAndServe(http_port, nil)
}

func main() {
	go grpcServer()
	go httpServer()

	// Wait forever
	select {}
}
