package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/kubeflow/katib/pkg/apis/manager/v1alpha3"
	suggestion "github.com/kubeflow/katib/pkg/suggestion/v1alpha3/goptuna"
	"google.golang.org/grpc"
	"k8s.io/klog"
)

var (
	port, host string
)

func main() {
	flag.StringVar(&port, "port", "6789", "the port to listen to for incoming HTTP connections")
	flag.StringVar(&host, "host", "0.0.0.0", "the host to listen to for incoming HTTP connections")
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		klog.Fatalf("Failed to listen: %v", err)
	}
	srv := grpc.NewServer()
	api_v1_alpha3.RegisterSuggestionServer(srv, suggestion.NewSuggestionService())

	klog.Infof("Start Goptuna suggestion service: %s:%s", host, port)
	err = srv.Serve(l)
	if err != nil {
		klog.Fatalf("Failed to serve: %v", err)
	}
	return
}
