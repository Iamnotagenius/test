// Database service for telegram bot and REST API
package main

import (
	"flag"
	"log"
	"net"
	"os"

	"github.com/Iamnotagenius/test/db/server"
	"github.com/Iamnotagenius/test/db/service"
	"github.com/go-pg/pg/v10"
	"google.golang.org/grpc"
)

var (
	serviceAddr = flag.String("service-addr", "localhost:50051", "The service address")
	dbAddr      = flag.String("db-addr", "localhost:5432", "The database address")
	dbUser      = flag.String("db-user", "postgres", "Database user")
)

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", *serviceAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	service.RegisterDatabaseTestServer(grpcServer, server.NewDatabaseServer(&pg.Options{
		User:     *dbUser,
		Addr:     *dbAddr,
		Password: os.Getenv("POSTGRESQL_PASSWORD"),
		Network:  "tcp",
	}))
	log.Printf("Server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
