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

var State eCriticalSystemState = RELEASED

var Port uint64

type Server struct {
	UnimplementedNodeServer
}

var logger *log.Logger

func main() {
	Port := ParseArguments(os.Args)
	logFile := InitLogging()
	defer ShutdownLogging(logFile)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	nodeServer := Server{}
	RegisterNodeServer(grpcServer, nodeServer)

	wait := make(chan struct{})
	go Serve(grpcServer, lis, wait)
	go ReadUserInput(wait)
	for {
		select {
		case <-wait:
			return
		}
	}
}

// ReadUserInput runs constantly, reading the standard input.
// Breaks out when the user types "quit" or "exit", and ignores empty lines.
// Otherwise, calls [AccessCriticalResource].
func ReadUserInput(wait chan struct{}) {
	reader := bufio.NewScanner(os.Stdin)
	logger.Println("Node started. Enter messages to store in shared file:")
	for {
		reader.Scan()
		if reader.Err() != nil {
			logger.Fatalf("failed to call Read: %v", reader.Err())
		}
		text := reader.Text()
		if text == "quit" || text == "exit" {
			wait <- struct{}{}
			break
		}
		if text == "" {
			continue
		}
		AccessCriticalResource(text)
	}
}

// Serve begins the server/service, allowing clients to execute its remote-procedure call functions.
func Serve(server *grpc.Server, lis net.Listener, wait chan struct{}) {
	err := server.Serve(lis)
	if err != nil {
		wait <- struct{}{}
		logger.Fatalf("failed to serve: %v", err)
	}
	logger.Printf("Listening on %s.\n", lis.Addr())
}

/** AccessCriticalResource appends the given text string to the file called "shared.txt". */
func AccessCriticalResource(text string) {
	file, err := os.OpenFile("shared.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Fatalf("failed to open file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Fatalf("failed to close file: %v", err)
		}
	}(file)

	stringToWrite := fmt.Sprintf("%s: %s\n", time.Now().Format(time.DateTime), text)
	_, err = file.WriteString(stringToWrite)
	if err != nil {
		logger.Fatalf("failed to write string: %v", err)
	}
	logger.Printf("Wrote to critical resource.\n")
}

type eCriticalSystemState uint8

const (
	RELEASED eCriticalSystemState = iota
	WANTED   eCriticalSystemState = iota
	HELD     eCriticalSystemState = iota
)

// InitLogging creates the logging file and initialises the logger with the correct prefix.
func InitLogging() *os.File {
	filename := fmt.Sprintf("log-%d.txt", Port)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	prefix := fmt.Sprintf("Node %d: ", Port)
	logger = log.New(file, prefix, 0)
	return file
}

// ShutdownLogging closes the file which backs the logger.
func ShutdownLogging(writer *os.File) {
	logger.Println("Server shut down.")
	_ = writer.Close()
}

// ParseArguments reads the command-line arguments and returns the port number specified
// therein.
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
