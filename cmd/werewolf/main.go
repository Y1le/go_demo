package main

import (
	"log"
	"net"

	pb "liam/pkg/werewolf"
	"liam/services/werewolf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// 注册狼人杀服务
	werewolfService := werewolf.NewWerewolfServer()
	pb.RegisterWerewolfServiceServer(grpcServer, werewolfService)

	// 启动反射服务
	reflection.Register(grpcServer)

	log.Println("Werewolf gRPC server is running on port :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
