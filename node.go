package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

var LamportTime int64 = 0

var State eCriticalSystemState

var Port uint16

func main() {
	Port := ParseArguments(os.Args)

	State = RELEASED
	logFile, logger := InitLogging()
	defer ShutdownLogging(logFile, logger)

	wait := make(chan struct{})
	reader := bufio.NewScanner(os.Stdin)
	fmt.Printf("Node is assigned port #%d.\n", Port)
	fmt.Println("Node started. Enter messages to store in shared file:")
	go func() {
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
	}()
	for {
		select {
		case <-wait:
			return
		}
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
