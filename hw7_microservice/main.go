package main

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	// println("usage: go test -v")
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalln("can't listen port", err)
	}

	server := grpc.NewServer()

	RegisterBizServer(server, NewBizServerImpl())
	fmt.Println("Starting server at :8081")
	server.Serve(lis)
}
