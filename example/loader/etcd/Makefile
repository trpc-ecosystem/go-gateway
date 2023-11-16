GO = go
DTOOLS = dtools
TARGET = trpc-gateway
SERVER = sever

# Install etcd https://etcd.io/docs/v3.5/install/ and make sure to move it to the path
# Start etcd
etcd:
	etcd

# 设置路由配置
update_router:
	cat  ./conf/router.yaml | etcdctl put router_conf

server:
	cd ../upstream && go build -o ../bin/$(SERVER) && cd ..
	../bin/$(SERVER) -conf=../upstream/trpc_go.yaml

run:
	go build -o ./bin/$(TARGET)
	./bin/$(TARGET) -conf=./trpc_go.yaml
