/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-13 11:11
**/

package ssh

import (
	"io"
	"net"

	"github.com/lemoyxk/console"
)

func Tcp(local string, remote string) {
	var localClient, err = net.Listen("tcp", local)
	if err != nil {
		console.Error(err)
		return
	}

	for {

		localConn, err := localClient.Accept()
		if err != nil {
			break
		}

		remoteConn, err := net.Dial("tcp", remote)
		if err != nil {
			console.Error(err)
			continue
		}

		handle(localConn, remoteConn)
	}

	_ = localClient.Close()
}

func handle(localConn net.Conn, remoteConn net.Conn) {
	go func() {
		_, _ = io.Copy(localConn, remoteConn)
		_ = localConn.Close()
		_ = remoteConn.Close()
	}()
	go func() {
		_, _ = io.Copy(remoteConn, localConn)
		_ = localConn.Close()
		_ = remoteConn.Close()
	}()
}
