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

	"github.com/lemoyxk/console"
)

// type Handler struct {
// 	Scheme string
// 	List   []string
// }

// func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// if len(h.List) == 1 {
// 	r.Host = h.List[0]
// }
//
// if len(h.List) == 2 {
// 	r.Host = strings.ReplaceAll(r.Host, h.List[0], h.List[1])
// }
//
// var proxy = httputil.NewSingleHostReverseProxy(&url.URL{Scheme: h.Scheme, Host: r.Host})
//
// proxy.ServeHTTP(w, r)

// }

// func Http(l net.Listener) {
// if len(list) == 0 {
// 	console.Error("[-] no target")
// 	return
// }

// var handler = &Handler{Scheme: scheme, List: list}
//
// var server = http.Server{
// 	Handler: handler,
// }
//
// var err = server.Serve(l)
// if err != nil {
// 	console.Error(err)
// 	return
// }
// }

func Http(l net.Listener) error {

	console.Info("Http server listen on:", l.Addr().String())

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

	if hostPortURL.Opaque == "443" { // https访问
		h = hostPortURL.Scheme
		p = "443"
	} else { // http访问
		var index = strings.Index(hostPortURL.Host, ":")
		if index == -1 { // host不带端口， 默认80
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
