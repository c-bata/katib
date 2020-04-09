package main

import (
	"net"

	"github.com/kubeflow/katib/pkg/apis/manager/v1alpha3"
	suggestion "github.com/kubeflow/katib/pkg/suggestion/v1alpha3/goptuna"
	"google.golang.org/grpc"
	"k8s.io/klog"
)

const address = "0.0.0.0:6789"

func main() {
	l, err := net.Listen("tcp", address)
	if err != nil {
		klog.Fatalf("Failed to listen: %v", err)
	}
	srv := grpc.NewServer()
	api_v1_alpha3.RegisterSuggestionServer(srv, suggestion.NewSuggestionService())

	klog.Infof("Start Goptuna suggestion service: %s", address)
	err = srv.Serve(l)
	if err != nil {
		klog.Fatalf("Failed to serve: %v", err)
	}
	return
}
