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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	http_port = ":50052"
	grpc_port = ":50053"
)

var (
	lskpmcs = make(map[string]string)
	xafis   = make(map[string]string)
	streams = make(map[string]map[pb.TraService_SubscribeServer]struct{})
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedTraServiceServer
}

func (s *server) Create(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	in_type := in.GetType()
	req_data := in.GetCreateRequest().GetData()
	log.Printf("GRPC Create Received from %s : %v", p.Addr.String(), req_data)

	switch in_type {
	case "lskpmc":
		for k, v := range req_data {
			lskpmcs[k] = v
		}
	case "xafi":
		for k, v := range req_data {
			xafis[k] = v
		}
	}

	// Once there are update, notify all subscribers
	change_chan <- in_type

	return &pb.TraServiceResponse{Type: in_type, Ret: 0, Response: &pb.TraServiceResponse_CreateResponse{CreateResponse: &pb.CreateResponse{}}}, nil
}

func (s *server) Update(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	in_type := in.GetType()
	req_data := in.GetCreateRequest().GetData()
	log.Printf("GRPC Update Received from %s : %v", p.Addr.String(), req_data)

	switch in_type {
	case "lskpmc":
		for k, v := range req_data {
			lskpmcs[k] = v
		}
	case "xafi":
		for k, v := range req_data {
			xafis[k] = v
		}
	}

	// Once there are update, notify all subscribers
	change_chan <- in_type

	return &pb.TraServiceResponse{Type: in_type, Ret: 0, Response: &pb.TraServiceResponse_UpdateResponse{UpdateResponse: &pb.UpdateResponse{}}}, nil
}

func (s *server) Retrieve(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	in_type := in.GetType()
	key := in.GetRetrieveRequest().GetKey()

	switch in_type {
	case "lskpmc":
		log.Printf("GRPC Retrieve Received from %s : %v=%v", p.Addr.String(), key, lskpmcs[key])

		return &pb.TraServiceResponse{Type: in_type, Ret: 0, Response: &pb.TraServiceResponse_RetrieveResponse{RetrieveResponse: &pb.RetrieveResponse{Data: map[string]string{key: lskpmcs[key]}}}}, nil
	case "xafi":
		log.Printf("GRPC Retrieve Received from %s : %v=%v", p.Addr.String(), key, xafis[key])

		return &pb.TraServiceResponse{Type: in_type, Ret: 0, Response: &pb.TraServiceResponse_RetrieveResponse{RetrieveResponse: &pb.RetrieveResponse{Data: map[string]string{key: xafis[key]}}}}, nil
	case "skey":
		value := query_skey(key)
		log.Printf("GRPC Retrieve Received from %s : %v=%v", p.Addr.String(), key, value)

		return &pb.TraServiceResponse{Type: in_type, Ret: 0, Response: &pb.TraServiceResponse_RetrieveResponse{RetrieveResponse: &pb.RetrieveResponse{Data: map[string]string{key: value}}}}, nil
	default:
		return &pb.TraServiceResponse{Type: in_type, Ret: -1, Reason: "Invalid Type", Response: &pb.TraServiceResponse_RetrieveResponse{RetrieveResponse: &pb.RetrieveResponse{Data: map[string]string{}}}}, nil
	}

}

func (s *server) Delete(ctx context.Context, in *pb.TraServiceRequest) (*pb.TraServiceResponse, error) {
	p, _ := peer.FromContext(ctx)

	in_type := in.GetType()
	key := in.GetDeleteRequest().GetKey()
	log.Printf("GRPC Delete Received from %s : %v", p.Addr.String(), key)

	switch in_type {
	case "lskpmc":
		delete(lskpmcs, key)
	case "xafi":
		delete(xafis, key)
	}

	return &pb.TraServiceResponse{Type: in_type, Ret: 0, Response: &pb.TraServiceResponse_DeleteResponse{DeleteResponse: &pb.DeleteResponse{}}}, nil
}

func (s *server) Subscribe(in *pb.TraServiceRequest, stream pb.TraService_SubscribeServer) error {
	p, _ := peer.FromContext(stream.Context())
	peer_addr := p.Addr.String()
	log.Printf("GRPC Subscribe Recieved from %s", peer_addr)

	in_type := in.GetType()

	if (streams[in_type] == nil) {
		streams[in_type] = make(map[pb.TraService_SubscribeServer]struct{})
	}

	streams[in_type][stream] = struct{}{}

	change_chan <- in_type
	for {
		if err := stream.Context().Err(); err != nil {
			log.Printf("remove %s from stream", p.Addr.String())
			delete(streams[in_type], stream)
			break
		}
		time.Sleep(time.Second)
	}
	return nil
}

var change_chan = make(chan string)

func (s *server) Notify() error {
	for {
		change_type := <-change_chan
		log.Printf("GRPC Notify %+v", streams[change_type])
		for stream, _ := range streams[change_type] {
			if err := stream.Context().Err(); err != nil {
				delete(streams[change_type], stream)
				continue
			}

			switch change_type {
			case "lskpmc":
				resp := pb.TraServiceResponse{Ret: 0, Response: &pb.TraServiceResponse_SubscribeResponse{SubscribeResponse: &pb.SubscribeResponse{Data: lskpmcs}}}
				stream.Send(&resp)
			case "xafi":
				resp := pb.TraServiceResponse{Ret: 0, Response: &pb.TraServiceResponse_SubscribeResponse{SubscribeResponse: &pb.SubscribeResponse{Data: lskpmcs}}}
				stream.Send(&resp)
			}

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

		change_chan <- "lskpmc"

		w.Header().Set("Content-Type", "application/json")
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

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Write(b)
	} else if r.Method == "DELETE" {
		// clear all lskpmcs
		lskpmcs = make(map[string]string)
		log.Printf("%s clear all lskpmcs: %#v", r.Method, lskpmcs)
		change_chan <- "lskpmc"
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	}
}

func xafisHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Read failed:", err)
		}
		defer r.Body.Close()

		err = json.Unmarshal(b, &xafis)
		if err != nil {
			log.Printf("json format error:", err)
		}
		log.Printf("%s update_data: %#v", r.Method, xafis)

		change_chan <- "xafi"

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	} else if r.Method == "GET" {
		b, err := json.Marshal(xafis)
		if err != nil {
			log.Printf("json format error:", err)
			return
		}

		log.Printf("%s  get_data: %#v", r.Method, xafis)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Write(b)
	} else if r.Method == "DELETE" {
		// clear all xafis
		xafis = make(map[string]string)
		log.Printf("%s clear all xafis: %#v", r.Method, lskpmcs)
		change_chan <- "xafi"
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	}
}

func sipInternalNodesHandler(w http.ResponseWriter, r *http.Request) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	tas_tra_srv, err := clientset.CoreV1().Services("cncs").Get(context.TODO(), "stas-stdn", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("No tas-tra service found")
	} else {
		clusterIp := tas_tra_srv.Spec.ClusterIP
		w.Write([]byte("[{\"fqdn\":\"stas-stdn.cncs.svc.cluster.local\",\"nodes\":[{\"nodeid\":0,\"ip\":\"" + clusterIp + "\",\"sipport\":5060,\"weight\":100}]}]"))
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	return
}

func httpServer() {
	log.Printf("Starting HTTP Server %s", http_port)
	http.HandleFunc("/lskpmcs", lskpmcsHandler)
	http.HandleFunc("/xafis", xafisHandler)
	http.HandleFunc("/SIP/INT/nodes", sipInternalNodesHandler)
	http.ListenAndServe(http_port, nil)
}

func init() {
}

func main() {
	go grpcServer()
	go httpServer()

	// Wait forever
	select {}
}
