GO = go
DTOOLS = dtools
TARGET = trpc-gateway
SERVER = sever

server:
	go run upstream/main.go

run:
	go build -o ./bin/$(TARGET)
	./bin/$(TARGET) -conf=./trpc_go.yaml  -router=./conf/router.yaml
