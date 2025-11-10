package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"

	. "DistEx/grpc"
)

var LamportTime int64 = 0

var State eCriticalSystemState

var Port uint16

type Server struct {
	UnimplementedNodeServer
}

func main() {
	Port := ParseArguments(os.Args)

	State = RELEASED
	logFile, logger := InitLogging()
	defer ShutdownLogging(logFile, logger)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	fmt.Printf("Listening on #%d.\n", lis.Addr())
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	nodeServer := Server{}
	RegisterNodeServer(grpcServer, nodeServer)

	wait := make(chan struct{})
	go Serve(grpcServer, lis, wait, logger)
	go ReadUserInput(wait)
	for {
		select {
		case <-wait:
			return
		}
	}
}

func ReadUserInput(wait chan struct{}) {
	reader := bufio.NewScanner(os.Stdin)
	fmt.Println("Node started. Enter messages to store in shared file:")
	for {
		reader.Scan()
		if reader.Err() != nil {
			log.Fatalf("fail to call Read: %v", reader.Err())
		}
		text := reader.Text()
		if text == "quit" || text == "exit" {
			wait <- struct{}{}
			break
		}
		if text == "" {
			continue
		}
		accessCriticalSystem(text)
	}
}

func Serve(server *grpc.Server, lis net.Listener, wait chan struct{}, logger *log.Logger) {
	err := server.Serve(lis)
	if err != nil {
		wait <- struct{}{}
		logger.Fatalf("failed to serve: %v", err)
	}
}

func accessCriticalSystem(text string) {
	file, err := os.OpenFile("shared.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("fail to open file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalf("fail to close file: %v", err)
		}
	}(file)

	_, err = file.WriteString(time.Now().Format(time.DateTime) + ": " + text + "\n")
	if err != nil {
		log.Fatalf("fail to write string: %v", err)
	}
}

type eCriticalSystemState uint8

const (
	RELEASED eCriticalSystemState = iota
	WANTED   eCriticalSystemState = iota
	HELD     eCriticalSystemState = iota
)

func InitLogging() (*os.File, *log.Logger) {
	file, err := os.Create("log.txt")
	if err != nil {
		log.Fatal(err)
	}
	return file, log.New(file, "Node _: ", 0)
}

func ShutdownLogging(writer *os.File, logger *log.Logger) {
	logger.Println("Server shut down.")
	_ = writer.Close()
}

func ParseArguments(args []string) (port uint64) {
	if len(args) != 2 {
		log.Fatal("Usage: go run main.go <port>")
	}
	port, err := strconv.ParseUint(args[1], 10, 16)
	if err != nil {
		log.Fatal(err)
	}
	return port
}
