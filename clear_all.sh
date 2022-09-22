# 终止全部enclave
sudo nitro-cli terminate-enclave --all
# 清除全部docker镜像
docker rm $(docker ps -a -q)
docker image rm proxy_server 
