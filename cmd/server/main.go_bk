/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a server for Greeter service.
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"encoding/json"
	pb "github.com/durd07/grpc-test/tra"
	"google.golang.org/grpc"
	"io/ioutil"
	"net/http"
)

const (
	port = ":50053"
)

var (
	streamCache = make(map[string] map[pb.TraService_SubscribeServer]struct{})
)

type watchInterface interface {
	add(obj interface{})
	modify(obj interface{})
	delete(obj interface{})
}

type endpointAction struct {
}

func (s *endpointAction) add(obj interface{}) {
	endpoints, ok := obj.(*corev1.Endpoints)
	if !ok {
		panic("Could not cast to Endpoint")
	}

	fqdn := endpoints.ObjectMeta.Name + "." + endpoints.Namespace + ".svc.cluster.local"
	data[fqdn] = &pb.TraResponse{
		Fqdn: fqdn,
	}

	for _, address := range endpoints.Subsets[0].Addresses {
		data[fqdn].Nodes = append(data[fqdn].Nodes, &pb.Node{NodeId: address.IP, Ip: address.IP, SipPort: uint32(endpoints.Subsets[0].Ports[0].Port), Weight: 1})
	}
}

func (s *endpointAction) modify(obj interface{}) {
	endpoints, ok := obj.(*corev1.Endpoints)
	if !ok {
		panic("Could not cast to Endpoint")
	}

	fqdn := endpoints.ObjectMeta.Name + "." + endpoints.Namespace + ".svc.cluster.local"
	data[fqdn] = &pb.TraResponse{
		Fqdn: fqdn,
	}

	for _, address := range endpoints.Subsets[0].Addresses {
		data[fqdn].Nodes = append(data[fqdn].Nodes, &pb.Node{NodeId: address.IP, Ip: address.IP, SipPort: uint32(endpoints.Subsets[0].Ports[0].Port), Weight: 20})
	}
}

func (s *endpointAction) delete(obj interface{}) {
	endpoints, ok := obj.(*corev1.Endpoints)
	if !ok {
		panic("Could not cast to Endpoint")
	}

	fqdn := endpoints.ObjectMeta.Name + "." + endpoints.Namespace + ".svc.cluster.local"
	delete(data, fqdn)
}

func watch() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespace := "default"

	endpoint_action := endpointAction{}

	watch, err := clientset.CoreV1().Endpoints(namespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	events := watch.ResultChan()

	for {
		event, ok := <-events
		log.Printf("### Endpoint Event %v: %+v\n\n", event.Type, event.Object)

		if event.Type == "ADDED" {
			endpoint_action.add(event.Object)
			change_chan <- struct{}{}
		} else if event.Type == "MODIFIED" {
			endpoint_action.modify(event.Object)
			change_chan <- struct{}{}
		} else if event.Type == "DELETED" {
			endpoint_action.delete(event.Object)
			change_chan <- struct{}{}
		} else {
			log.Printf("Endpoint watch unhandled event: %v", event.Type)
			break
		}

		if !ok {
			log.Printf("should panic here")
		}
	}
}

type NodeData struct {
	Fqdn     string `json:"fqdn"`
	Node_id  string `json:"node_id"`
	Ip       string `json:"ip"`
	Sip_port uint32 `json:"sip_port"`
	Weight   uint32 `json:"weight"`
}

var (
	data = map[string]*pb.TraResponse{
//		"tafe.default.svc.cluster.local": &pb.TraResponse{
//			Fqdn: "tafe.svc.default.cluster.local",
//			Nodes: []*pb.Node{
//				&pb.Node{NodeId: "1", Ip: "192.168.0.1", SipPort: 5060, Weight: 50},
//				&pb.Node{NodeId: "2", Ip: "192.168.0.2", SipPort: 5060, Weight: 50},
//			},
//		},
//
//		"sips.default.svc.cluster.local": &pb.TraResponse{
//			Fqdn: "tafe.svc.default.cluster.local",
//			Nodes: []*pb.Node{
//				&pb.Node{NodeId: "1", Ip: "192.168.0.1", SipPort: 5060, Weight: 50},
//				&pb.Node{NodeId: "2", Ip: "192.168.0.2", SipPort: 5060, Weight: 50},
//			},
//		},
//
//		"sipc.default.svc.cluster.local": &pb.TraResponse{
//			Fqdn: "tafe.svc.default.cluster.local",
//			Nodes: []*pb.Node{
//				&pb.Node{NodeId: "1", Ip: "192.168.0.1", SipPort: 5060, Weight: 50},
//				&pb.Node{NodeId: "2", Ip: "192.168.0.2", SipPort: 5060, Weight: 50},
//			},
//		},
//
//		"scfe.default.svc.cluster.local": &pb.TraResponse{
//			Fqdn: "scfe.svc.default.cluster.local",
//			Nodes: []*pb.Node{
//				&pb.Node{NodeId: "3", Ip: "192.168.0.3", SipPort: 5060, Weight: 50},
//				&pb.Node{NodeId: "4", Ip: "192.168.0.4", SipPort: 5060, Weight: 50},
//			},
//		},
	}
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedTraServiceServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) Nodes(ctx context.Context, in *pb.TraRequest) (*pb.TraResponse, error) {
	log.Printf("Received: %v", in.Fqdn)
	var response = data[in.Fqdn]
	log.Println(response)
	return response, nil
	//	return &pb.TraResponse{Fqdn: in.Fqdn, Nodes: []*pb.Node{
	//			&pb.Node{NodeId: "1", Ip: "192.168.0.1", SipPort: 5060, Weight: 50},
	//			&pb.Node{NodeId: "2", Ip: "192.168.0.2", SipPort: 5060, Weight: 50},
	//	}}, nil
}

func (s *server) Subscribe(in *pb.TraRequest, stream pb.TraService_SubscribeServer) error {
	log.Printf("Received Subscribe: %v", in.Fqdn)

	if _, ok := streamCache[in.Fqdn]; !ok {
		memember := make(map[pb.TraService_SubscribeServer] struct{})
		memember[stream] = struct{}{}
		streamCache[in.Fqdn] = memember
	} else {
		streamCache[in.Fqdn][stream] = struct{}{}
	}

	change_chan <- struct{}{}
	for {
		if err := stream.Context().Err(); err != nil {
			delete(streamCache[in.Fqdn], stream)
			break
		}
	}
	return nil
}

var change_chan = make(chan struct{})

func (s *server) Notify() error {
	for {
		_ = <-change_chan
		log.Printf("XXXXXX do notify %v\n", streamCache)
		for k, stream_list := range streamCache {
			for stream, _ := range stream_list {
				if err := stream.Context().Err(); err != nil {
					delete(streamCache[k], stream)
					continue
				}
				stream.Send(data[k])
				log.Printf("send to %+v\n", stream)
			}
		}
	}
}

func grpcServer() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	myserver := server{}
	go myserver.Notify()

	pb.RegisterTraServiceServer(s, &myserver)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func updateNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		log.Println("XXX")

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Read failed:", err)
		}
		defer r.Body.Close()

		var update_data []NodeData
		err = json.Unmarshal(b, &update_data)
		if err != nil {
			log.Println("json format error:", err)
		}
		log.Println("update_data:", update_data)

		data = make(map[string]*pb.TraResponse)
		for _, v := range update_data {
			if _, ok := data[v.Fqdn]; ok {
				data[v.Fqdn].Nodes = append(data[v.Fqdn].Nodes, &pb.Node{NodeId: v.Node_id, Ip: v.Ip, SipPort: v.Sip_port, Weight: v.Weight})
			} else {
				data[v.Fqdn] = &pb.TraResponse{
					Fqdn: v.Fqdn,
					Nodes: []*pb.Node{
						&pb.Node{NodeId: v.Node_id, Ip: v.Ip, SipPort: v.Sip_port, Weight: v.Weight},
					},
				}
			}
		}

		for k, v := range data {
			log.Println(k, v.Nodes)
		}

		change_chan <- struct{}{}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	} else if r.Method == "GET" {
		var tmp []NodeData
		for _, v := range data {
			for _, n := range v.Nodes {
				tmp = append(tmp, NodeData{v.Fqdn, n.NodeId, n.Ip, n.SipPort, n.Weight})
			}
		}
		log.Println(tmp)
		b, err := json.Marshal(tmp)
		if err != nil {
			log.Println("json format error:", err)
			return
		}

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
	http.HandleFunc("/update", updateNodes)
	http.ListenAndServe(":50052", nil)
}

func main() {
	go watch()
	go grpcServer()
	httpServer()
}
