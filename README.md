# mq-proxy

***前提条件：*** 需要一个rabbitmq服务，或者使用脚本启动一个rabbitmq的docker容器，`./start-docker-rabbit.sh`

## 运行说明

mq_proxy能够接收一个名为target的参数。当target等于client时，表示启动运行在enclave外部宿主机上的程序；当target等于server时，表示启动运行在enclave内部的程序。

当target为client时，还需要另外两个参数cid和port；分别表示enclave或者容器的cid和vsock服务的端口号。

```
# 在宿主机上
mq_proxy --target=client --cid=3 --port=3000

# 在enclave或者容器内
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

## KMS加密代理

*** 前提条件 *** 需要在支持Nitro Enclave的EC2服务器上测试

1. 在开发环境中，先运行`build.sh`

2. 然后运行`upload.sh`，将需要的文件上传至服务器

3. 登陆到服务器

4. 在服务器上运行`build-coker-image.sh`创建docker镜像

5. 再运行`build-eif.sh`创建eif文件

6. 执行`create_enclave.sh`创建并启动enclave

7. 打开enclave的console，`nitro-cli console --enclave-id`

8. 在另外一个登陆了服务器的终端中，执行`mq_proxy --target=client --cid=enclave的cid --port=3000`

enclave在收到proxy准备就绪后，除了执行之前广播和p2p消息外，会请求kms代理加密字符串“Hello world!”，后台可以看到加密后的cipher字符串。

另，`clear_all.sh`可以清除存在的enclave和docker中proxy_server镜像。