package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/eiannone/keyboard"
	pb "github.com/michalskalski/greeter/proto" // Replace with the actual path to your generated proto files
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
)

// Server is used to implement the Greeter service.
type server struct {
	name string
	pb.UnimplementedGreeterServer
}

// Ping implements the Unary RPC, responding with "Pong".
func (s *server) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	// log from who the request came
	if peer, ok := peer.FromContext(ctx); ok {
		log.Printf("Received Ping request from %s", peer)
	}
	return &pb.PingResponse{Message: "Pong from " + s.name}, nil
}

// StreamPong implements the Bidirectional Streaming RPC, responding with "Pong" for each "Ping".
func (s *server) StreamPong(stream pb.Greeter_StreamPongServer) error {
	for {
		// Receive a "Ping" message from the client.
		_, err := stream.Recv()
		if err == io.EOF {
			return nil // Client has finished sending messages.
		}
		if err != nil {
			return err
		}

		if peer, ok := peer.FromContext(stream.Context()); ok {
			log.Printf("Received Ping request from %s", peer)
		}
		// Respond with "Pong".
		response := &pb.PingResponse{
			Message: fmt.Sprintf("Pong from %s %s", s.name, time.Now().String()),
		}
		if err := stream.Send(response); err != nil {
			return err
		}
	}
}

// runServer initializes and starts the gRPC server.
func runServer() {
	// Get the hostname to use as part of the server name.
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}
	name := hostname
	envName := os.Getenv("ENV_NAME")
	if envName != "" {
		name = fmt.Sprintf("%s.%s", name, envName)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{name: name})
	log.Printf("Server is listening on port 50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// runClient connects to the server and demonstrates both Unary and Bidirectional Streaming RPCs.
func runClient(address string, insecureConnection bool) {
	clientCredentials := credentials.NewTLS(&tls.Config{})
	if insecureConnection {
		clientCredentials = insecure.NewCredentials()
	}
	// Connect to the server.
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(clientCredentials))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewGreeterClient(conn)

	// Unary RPC Example
	pingResponse, err := client.Ping(context.Background(), &pb.PingRequest{})
	if err != nil {
		log.Fatalf("Error calling Ping: %v", err)
	}
	log.Printf("Unary Ping Response: %s", pingResponse.Message)

	// Bidirectional Streaming RPC Example
	stream, err := client.StreamPong(context.Background())
	if err != nil {
		log.Fatalf("Error creating StreamPong stream: %v", err)
	}
	defer stream.CloseSend()

	// Start a goroutine to receive messages from the server.
	go func() {
		for {
			in, err := stream.Recv()
			if err != nil {
				log.Printf("Error receiving message: %v", err)
				return
			}
			log.Printf("Server: %s", in.Message)
		}
	}()

	// Open the keyboard listener for user input.
	if err := keyboard.Open(); err != nil {
		log.Fatalf("failed to open keyboard listener: %v", err)
	}
	defer keyboard.Close()

	log.Println("Press 'p' to send a Ping, or 'e' to exit.")

	// Capture keypresses to control the client behavior.
	for {
		char, key, err := keyboard.GetSingleKey()
		if err != nil {
			log.Fatalf("Error reading key: %v", err)
		}

		switch char {
		case 'p':
			if err := stream.Send(&pb.PingRequest{}); err != nil {
				log.Printf("Error sending Ping: %v", err)
				return
			}
		case 'e':
			log.Println("Exiting client.")
			return
		default:
			if key == keyboard.KeyEsc {
				log.Println("Exiting client.")
				return
			}
			log.Println("Invalid input. Press 'p' to send a Ping, or 'e' to exit.")
		}
	}
}

func main() {
	// Define command-line flags to control behavior.
	mode := flag.String("mode", "server", "Mode to run: server or client")
	address := flag.String("address", "localhost:50051", "The server address in the format host:port")
	insecureConnection := flag.Bool("insecure", false, "Use an insecure connection (client only)")
	flag.Parse()

	// Run as either server or client based on the flag.
	if *mode == "server" {
		runServer()
	} else if *mode == "client" {
		runClient(*address, *insecureConnection)
	} else {
		log.Fatalf("Invalid mode: %s. Use 'server' or 'client'.", *mode)
	}
}
