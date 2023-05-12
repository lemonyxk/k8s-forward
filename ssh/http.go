/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-13 11:30
**/

package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"

	"github.com/lemonyxk/console"
)

func Http(l net.Listener) error {

	console.Info("http server listen on:", l.Addr().String())

	for {
		client, err := l.Accept()
		if err != nil {
			break
		}

		go httpHandler(client)
	}

	_ = l.Close()

	return errors.New("http server stopped")
}

func httpHandler(client net.Conn) {

	var b [4096]byte
	n, err := client.Read(b[:])
	if err != nil {
		_ = client.Close()
		console.Error(err)
		return
	}
	var method, host, address string
	var index = bytes.IndexByte(b[:], '\n')
	if index == -1 {
		_ = client.Close()
		return
	}

	_, _ = fmt.Sscanf(string(b[:index]), "%s%s", &method, &host)
	hostPortURL, err := url.Parse(host)
	if err != nil {
		_ = client.Close()
		console.Error(err)
		return
	}

	var h string
	var p string

	if hostPortURL.Opaque == "443" {
		h = hostPortURL.Scheme
		p = "443"
	} else {
		var index = strings.Index(hostPortURL.Host, ":")
		if index == -1 {
			h = hostPortURL.Host
			p = "80"
		} else {
			h = hostPortURL.Host[:index]
			p = "80"
		}
	}

	address = h + ":" + p

	server, err := net.Dial("tcp", address)
	if err != nil {
		_ = client.Close()
		console.Error(err)
		return
	}

	if method == "CONNECT" {
		_, _ = fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\nProxy-agent: Pyx\r\n\r\n")
	} else {
		_, _ = server.Write(b[:n])
	}

	go func() {
		_, _ = io.Copy(client, server)
		_ = server.Close()
		_ = client.Close()
	}()

	go func() {
		_, _ = io.Copy(server, client)
		_ = server.Close()
		_ = client.Close()
	}()
}
