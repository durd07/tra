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
func (s *server) UpdateLskpmc(ctx context.Context, in *pb.LskpmcRequest) (int32, error) {
	p, _ := peer.FromContext(ctx)
	klog.Infof("GRPC UpdateLskpmc Received from %s : %v", p.Addr.String(), in.lskpmcs)

	for l := range in.lskpmcs {
		lskpmcs[l.key] = l.val
	}

	return 0, nil
}

func (s *server) GetIpFromLskpmc(ctx context.Context, in string) (string, error) {
	p, _ := peer.FromContext(ctx)
	klog.Infof("GRPC GetIpFromLskpmc Received from %s : %v", p.Addr.String(), in)
	return lskpmcs[in], nil
}

func (s *server) Subscribe(in *pb.LskpmcRequest, stream pb.TraService_SubscribeServer) error {
	p, _ := peer.FromContext(stream.Context())
	peer_addr := p.Addr.String()
	klog.Infof("GRPC Subscribe Recieved from %s", peer_addr)

	streams[stream] = struct{}

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
		klog.Infof("GRPC Notify %+v\n", streamCache)
		for stream, _ := range streams {
			if err := stream.Context().Err(); err != nil {
				delete(streams, stream)
				continue
			}
			stream.Send(lskpmcs)

			p, _ := peer.FromContext(stream.Context())
			klog.Infof("     send to %+v\n", p.Addr.String())
		}
	}
}

func grpcServer() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	myserver := server{}
	go myserver.Notify()

	pb.RegisterTraServiceServer(s, &myserver)
	if err := s.Serve(lis); err != nil {
		klog.Fatalf("failed to serve: %v", err)
	}
}


func lskpmcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			klog.Infoln("Read failed:", err)
		}
		defer r.Body.Close()

		err = json.Unmarshal(b, &lskpmcs)
		if err != nil {
			klog.Infoln("json format error:", err)
		}
		klog.Infof("%s update_data: %#v", r.Method, lskpmcs)

		change_chan <- struct{}{}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	} else if r.Method == "GET" {
		b, err := json.Marshal(lskpmcs)
		if err != nil {
			klog.Infoln("json format error:", err)
			return
		}

		klog.Infof("%s get_data: %#v", r.Method, lskpmcs)

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
	http.HandleFunc("/lskpmc", lskpmcHandler)
	http.ListenAndServe(":50052", nil)
}


func main() {
	go grpcServer()
	go httpServer()

	// Wait forever
	select {}
}
