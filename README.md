# mq-proxy

***前提条件：*** 需要一个rabbitmq服务，或者使用脚本启动一个rabbitmq的docker容器，`./start-docker-rabbit.sh`

## 运行说明

mq_proxy能够接收一个名为target的参数。当target等于client时，表示启动运行在enclave外部宿主机上的程序；当target等于server时，表示启动运行在enclave内部的程序。

当target为client时，还需要另外两个参数cid和port；本别表示enclave或者容器的cid和vsock服务的端口号。

```
# 在宿主机上
mq_proxy --target=client --cid=3 --port=3000

# 在enclave内
mq_proxy --target=server

```

启动顺序：先启动enclave或者容器内部的server程序，再启动宿主机上的client。

如果在mac上调试开发，可以使用`go mq_proxy.go --target=client --cid=3 --port=3000`启动宿主机上的程序。

而创建docker的镜像，请执行如下脚本

```
./build.sh
./clear_images.sh
./build-dockcer-image.sh
```

然后启动docker镜像，模拟enclave环境，请执行如下脚本

```
./start-docker-proxy-server.sh
```

