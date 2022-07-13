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

func Tcp(l net.Listener, remote string) {
	console.Info("Tcp server listen on:", l.Addr().String())

	for {
		localConn, err := l.Accept()
		if err != nil {
			break
		}

		remoteConn, err := net.Dial("tcp", remote)
		if err != nil {
			console.Error(err)
			continue
		}

		tcpHandle(localConn, remoteConn)
	}

	_ = l.Close()

	console.Info("Tcp server closed")
}

func tcpHandle(localConn net.Conn, remoteConn net.Conn) {
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
