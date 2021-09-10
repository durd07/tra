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
	proto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	pb "github.com/durd07/tra/proto"
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

	m := pb.CreateLskpmcRequest{}
	err := anypb.UnmarshalTo(in.Request, &m, proto.UnmarshalOptions{})
	if err != nil {
		log.Fatalf("GRPC Create Unmarshal failed")
		return &pb.TraServiceResponse{ Ret: -1, Reason: err.Error()}, nil
	}

	key := m.Lskpmc.Key
	val := m.Lskpmc.Val
	log.Printf("GRPC Create Received from %s : %v=%v", p.Addr.String(), key, val)

	lskpmcs[key] = val

	// Once there are update, notify all subscribers
	change_chan <- struct{}{}

	return &pb.TraServiceResponse{ Ret: 0}, nil
}

func (s *server) UpdateLskpmc(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	m := pb.UpdateLskpmcRequest{}
	err := anypb.UnmarshalTo(in.Request, &m, proto.UnmarshalOptions{})
	if err != nil {
		log.Fatalf("GRPC Update Unmarshal failed")
		return &pb.TraServiceResponse{ Ret: -1, Reason: err.Error()}, nil
	}

	key := m.Lskpmc.Key
	val := m.Lskpmc.Val
	log.Printf("GRPC Update Received from %s : %v=%v", p.Addr.String(), key, val)

	lskpmcs[key] = val

	// Once there are update, notify all subscribers
	change_chan <- struct{}{}

	return &pb.TraServiceResponse{ Ret: 0}, nil
}

func (s *server) RetrieveLskpmc(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	m := pb.RetrieveLskpmcRequest{}
	err := anypb.UnmarshalTo(in.Request, &m, proto.UnmarshalOptions{})
	if err != nil {
		log.Fatalf("GRPC Retrieve Unmarshal failed")
		return &pb.TraServiceResponse{ Ret: -1, Reason: err.Error()}, nil
	}

	key := m.Lskpmc
	log.Printf("GRPC Retrieve Received from %s : %v", p.Addr.String(), key)

	// Once there are update, notify all subscribers
	change_chan <- struct{}{}

	any_resp, err := anypb.New(&pb.RetrieveLskpmcResponse{Lskpmc: &pb.Lskpmc{Key: key, Val: lskpmcs[key]}})
	return &pb.TraServiceResponse{ Ret: 0, Response: any_resp}, nil
}

func (s *server) DeleteLskpmc(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	m := pb.DeleteLskpmcRequest{}
	err := anypb.UnmarshalTo(in.Request, &m, proto.UnmarshalOptions{})
	if err != nil {
		log.Fatalf("GRPC Delete Unmarshal failed")
		return &pb.TraServiceResponse{ Ret: -1, Reason: err.Error()}, nil
	}

	key := m.Lskpmc
	log.Printf("GRPC Delete Received from %s : %v", p.Addr.String(), key)

	delete(lskpmcs, key)

	// Once there are update, notify all subscribers
	change_chan <- struct{}{}

	return &pb.TraServiceResponse{ Ret: 0}, nil
}

func (s *server) SubscribeLskpmc(in *pb.TraServiceRequest, stream pb.TraService_SubscribeLskpmcServer) error {
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

			r := pb.SubscribeLskpmcResponse{Lskpmcs: []*pb.Lskpmc{}}

			for k, v := range lskpmcs {
				lskpmc := pb.Lskpmc{Key :k, Val: v}
				r.Lskpmcs = append(r.Lskpmcs, &lskpmc)
			}

			any_resp, _ := anypb.New(&r)
			resp := pb.TraServiceResponse{Ret: 0, Response: any_resp}
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
