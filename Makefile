install: grpc/consensus_grpc.pb.go grpc/consensus.pb.go node.exe

grpc/consensus_grpc.pb.go grpc/consensus.pb.go: grpc/consensus.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative grpc/consensus.proto

proto: grpc/consensus_grpc.pb.go grpc/consensus.pb.go

node.exe: node.go
	go build -o node.exe node.go