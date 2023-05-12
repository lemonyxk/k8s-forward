/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-12 15:25
**/

package ssh

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/lemonyxk/console"
)

func Socks5(l net.Listener) error {
	console.Info("cocks5 server listen on:", l.Addr().String())

	for {
		client, err := l.Accept()
		if err != nil {
			break
		}

		go socks5Handler(client)
	}

	_ = l.Close()

	return errors.New("socks5 server stopped")
}

func socks5Handler(client net.Conn) {
	if err := Socks5Auth(client); err != nil {
		console.Error(err)
		_ = client.Close()
		return
	}

	target, err := Socks5Connect(client)
	if err != nil {
		console.Error(err)
		_ = client.Close()
		return
	}

	Socks5Forward(client, target)
}

func Socks5Auth(client net.Conn) (err error) {
	buf := make([]byte, 256)

	// 读取 VER 和 N_METHODS
	n, err := io.ReadFull(client, buf[:2])
	if n != 2 {
		return errors.New("reading header: " + err.Error())
	}

	ver, nMethods := int(buf[0]), int(buf[1])
	if ver != 5 {
		return errors.New("invalid version")
	}

	// 读取 METHODS 列表
	n, err = io.ReadFull(client, buf[:nMethods])
	if n != nMethods {
		return errors.New("reading methods: " + err.Error())
	}

	// 无需认证
	n, err = client.Write([]byte{0x05, 0x00})
	if n != 2 || err != nil {
		return errors.New("write rsp: " + err.Error())
	}

	return nil
}

func Socks5Connect(client net.Conn) (net.Conn, error) {
	buf := make([]byte, 256)

	n, err := io.ReadFull(client, buf[:4])
	if n != 4 {
		return nil, errors.New("read header: " + err.Error())
	}

	ver, cmd, _, aTyp := buf[0], buf[1], buf[2], buf[3]
	if ver != 5 || cmd != 1 {
		return nil, errors.New("invalid ver/cmd")
	}

	addr := ""
	switch aTyp {
	case 1:
		n, err = io.ReadFull(client, buf[:4])
		if n != 4 {
			return nil, errors.New("invalid IPv4: " + err.Error())
		}
		addr = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])

	case 3:
		n, err = io.ReadFull(client, buf[:1])
		if n != 1 {
			return nil, errors.New("invalid hostname: " + err.Error())
		}
		addrLen := int(buf[0])

		n, err = io.ReadFull(client, buf[:addrLen])
		if n != addrLen {
			return nil, errors.New("invalid hostname: " + err.Error())
		}
		addr = string(buf[:addrLen])

	case 4:
		return nil, errors.New("IPv6: no supported yet")

	default:
		return nil, errors.New("invalid atyp")
	}

	n, err = io.ReadFull(client, buf[:2])
	if n != 2 {
		return nil, errors.New("read port: " + err.Error())
	}

	port := binary.BigEndian.Uint16(buf[:2])
	dstAddrPort := fmt.Sprintf("%s:%d", addr, port)

	dst, err := net.Dial("tcp", dstAddrPort)
	if err != nil {
		return nil, errors.New("dial dst: " + err.Error())
	}

	n, err = client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		_ = dst.Close()
		return nil, errors.New("write rsp: " + err.Error())
	}

	return dst, nil
}

func Socks5Forward(client, target net.Conn) {
	forward := func(src, dst net.Conn) {
		_, _ = io.Copy(src, dst)
		_ = src.Close()
		_ = dst.Close()
	}
	go forward(client, target)
	go forward(target, client)
}
