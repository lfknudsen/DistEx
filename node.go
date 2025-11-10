package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	. "DistEx/grpc"
)

var Nodes []int64

var LamportTime *int64

var TimeLock sync.Mutex = sync.Mutex{}

var State eCriticalSystemState = RELEASED

type eCriticalSystemState uint8

const (
	RELEASED eCriticalSystemState = iota
	WANTED   eCriticalSystemState = iota
	HELD     eCriticalSystemState = iota
)

var Port *int64

var RequestQueue []int64

type Server struct {
	UnimplementedNodeServer
}

var logger *log.Logger

var wg sync.WaitGroup

func main() {
	LamportTime = new(int64)
	Port = ParseArguments(os.Args)
	setupOtherNodeList()
	filename := fmt.Sprintf("log-%d.txt", *Port)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	prefix := fmt.Sprintf("Node %d: ", *Port)
	logger = log.New(file, prefix, 0)
	defer ShutdownLogging(file)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *Port))
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
	fmt.Printf("Node %d started. Enter messages to store in shared file:\n", *Port)
	logf("Node has begun listening to user input through standard input.")
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
	logf("Listening on %s.\n", lis.Addr())
}

// AccessCriticalResource appends the given text string to the file called "shared.txt". */
func AccessCriticalResource(text string) {
	enter()

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

	stringToWrite := fmt.Sprintf("%s: T: %d; p:%d; %s\n",
		time.Now().Format(time.DateTime),
		*LamportTime,
		*Port,
		text)
	_, err = file.WriteString(stringToWrite)
	if err != nil {
		logger.Fatalf("failed to write string: %v", err)
	}
	logf("Wrote to critical resource: \"%s\"\n", text)
	exit()
}

// enter performs the necessary actions when attempting to gain access to the
// critical section.
func enter() {
	IncrementTime()
	State = WANTED
	logf("Setting state to WANTED.")
	logf("Broadcasting to all other nodes.")
	for _, node := range Nodes {
		logf("Broadcast to node %d\n", node)
		wg.Add(1) //increment waitgroup for the routines to finish themselves
		go broadcast(node)
	}

	wg.Wait() // added wait condition (meaning N-1 have been closed)
	State = HELD
	logf("Received N-1 replies. Setting state to HELD.")
}

// broadcast sends a request-for-access to the given port.
func broadcast(targetPort int64) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	targetAddress := fmt.Sprintf("localhost:%d", targetPort)
	conn, err := grpc.NewClient(targetAddress, opts...)
	if err != nil {
		logger.Fatalf("Failed to dial: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err = conn.Close()
		if err != nil {
			logger.Fatalf("Error closing connection:\n%v", err)
		}
	}(conn)
	client := NewNodeClient(conn)
	_, err = client.Request(context.Background(), &Message{
		Timestamp: LamportTime,
		Id:        Port,
	})
	// If the client did not exist, we decrement the wait group instead of waiting
	// for a response that will never arrive.
	if err != nil {
		wg.Done()
		logf("Failed to broadcast to node %d. Skipping it.", targetPort)
	} else {
		logf("Broadcasted to node %d.", targetPort)
	}
}

// ShutdownLogging closes the file which backs the logger.
func ShutdownLogging(writer *os.File) {
	logf("Node shut down.\n")
	_ = writer.Close()
}

// ParseArguments reads the command-line arguments and returns the port number specified
// therein.
func ParseArguments(args []string) *int64 {
	if len(args) != 2 {
		log.Fatal("Usage: go run main.go <port>")
	}
	port, err := strconv.ParseInt(args[1], 10, 16)
	if err != nil {
		log.Fatal(err)
	}
	return &port
}

// Request is the RPC executed when the caller wants access to the critical area.
func Request(msg *Message) *Reply {
	isBusy := false
	logf("Received critical area access request from node %d.", msg.GetId())
	if State == HELD || (State == WANTED && *LamportTime < *msg.Timestamp) {
		isBusy = true
	}
	UpdateTime(msg.Timestamp)
	IncrementTime()
	if isBusy {
		logf("Adding request to queue.")
		RequestQueue = append(RequestQueue, msg.GetId())
	} else {
		logf("Replying immediately to request.")
		reply(msg.GetId())
	}
	return &Reply{
		IsQueued: &isBusy,
	}
}

// Respond is the RPC executed when the caller is finished in the critical area.
func Respond(msg *Message) *Released {
	wg.Done()
	UpdateTime(msg.Timestamp)
	logf("Received reply to critical area access request from node %d.", msg.GetId())
	return &Released{}
}

// exit performs the necessary actions when leaving the critical section.
func exit() {
	State = RELEASED
	for _, targetPort := range RequestQueue {
		reply(targetPort)
	}
}

// reply sends the individual response to a target node after this node has finished
// inside the critical section.
func reply(targetPort int64) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	targetAddress := fmt.Sprintf("localhost:%d", targetPort)
	conn, err := grpc.NewClient(targetAddress, opts...)
	if err != nil {
		logger.Fatalf("Failed to dial: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err = conn.Close()
		if err != nil {
			logger.Fatalf("Error closing connection:\n%v", err)
		}
	}(conn)
	client := NewNodeClient(conn)
	_, err = client.Respond(context.Background(), &Message{
		Timestamp: LamportTime,
		Id:        Port,
	})
}

// logf writes a message to the log file.
func logf(format string, v ...any) {
	prefix := fmt.Sprintf("Node %d. Time: %d. ", *Port, *LamportTime)
	logger.SetPrefix(prefix)
	text := fmt.Sprintf(format, v...)
	if !(strings.HasSuffix(format, "\n") || strings.HasSuffix(format, "\r")) {
		logger.Println(text)
	} else {
		logger.Print(text)
	}
}

// setupOtherNodeList creates the list of other distributed nodes.
func setupOtherNodeList() {
	Nodes := []int64{5000, 5001, 5002}

	// Remove own node from list
	for i := range Nodes {
		if Nodes[i] == *Port {
			if i == len(Nodes)-1 {
				Nodes = Nodes[:i]
			} else {
				Nodes = append(Nodes[:i], Nodes[i+1:]...)
			}
			break
		}
	}
}

func IncrementTime() {
	TimeLock.Lock()
	defer TimeLock.Unlock()
	*LamportTime += 1
}

func UpdateTime(other *int64) {
	TimeLock.Lock()
	defer TimeLock.Unlock()
	if *LamportTime < *other {
		*LamportTime = *other
	}
}
