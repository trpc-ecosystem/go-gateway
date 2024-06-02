TARGET = trpc-gateway
SERVER = sever

server:
	cd ../upstream && go build -o ../bin/$(SERVER) && cd ..
	../bin/$(SERVER) -conf=../upstream/trpc_go.yaml

run:
	go build -o ./bin/$(TARGET)
	./bin/$(TARGET) -conf=./trpc_go.yaml -router=./conf/router.yaml
