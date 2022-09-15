package mqproxy

import "github.com/holynull/my-vsock/my_vsock"

func Start() {
	my_vsock.ConnetctServer(2, 3000)
}
