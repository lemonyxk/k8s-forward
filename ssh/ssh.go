/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-08 22:44
**/

package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/lemonyxk/k8s-forward/tools"
	"github.com/lemoyxk/console"
	"golang.org/x/crypto/ssh"
)

func Server(user, password, host string, port int) (*ssh.Session, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		session      *ssh.Session
		err          error
	)

	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 3 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// connect to ssh
	addr = fmt.Sprintf("%s:%d", host, port)

	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create session
	if session, err = client.NewSession(); err != nil {
		return nil, err
	}

	return session, nil
}

func sshListen(sshClientConn *ssh.Client, remoteAddr string) (net.Listener, error) {
	l, err := sshClientConn.Listen("tcp", remoteAddr)
	if err != nil {
		// net broken, not closed
		if strings.HasSuffix(err.Error(), "tcpip-forward request denied by peer") {
			conn, e := sshClientConn.Dial("tcp", remoteAddr)
			if e != nil {
				return nil, e
			}
			// send a request to close the connection
			_, _ = conn.Write(nil)
			time.Sleep(time.Second)
			return sshListen(sshClientConn, remoteAddr)
		}
		return nil, err
	}

	return l, nil
}

func LocalForward(username, password, serverAddr, remoteAddr, localAddr string, args ...string) (chan struct{}, error) {
	// Setup SSH config (type *ssh.ClientConfig)
	config := &ssh.ClientConfig{
		User:    username,
		Auth:    []ssh.AuthMethod{ssh.Password(password)},
		Timeout: time.Second * 3,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// Setup sshClientConn (type *ssh.ClientConn)
	sshClientConn, err := ssh.Dial("tcp", serverAddr, config)
	if err != nil {
		return nil, err
	}

	// create session
	session, err := sshClientConn.NewSession()
	if err != nil {
		return nil, err
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// Setup localListener (type net.Listener)
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return nil, err
	}

	console.Info("LocalForward", localAddr, "to", remoteAddr)

	var closeFn = func() {
		_ = localListener.Close()
		_ = sshClientConn.Close()
		_ = session.Close()
		console.Info("Close LocalForward")
	}

	var stopChan = make(chan struct{}, 1)

	go func() {
		select {
		case <-stopChan:
			closeFn()
		}
	}()

	var ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				_, err := session.SendRequest(config.User, false, nil)
				if err != nil {
					closeFn()
					return
				}
			}
		}
	}()

	// --------

	var proxyMode = tools.GetArgs([]string{"proxy", "--proxy"}, args)
	switch proxyMode {
	case "socks5":
		// socks5 proxy
		l, err := sshListen(sshClientConn, remoteAddr)
		if err != nil {
			return nil, err
		}
		go Socks5(l)

	case "tcp":
		// tcp proxy
		var target = tools.GetArgs([]string{proxyMode}, args)
		if target != "" {
			l, err := sshListen(sshClientConn, remoteAddr)
			if err != nil {
				return nil, err
			}
			go Tcp(l, target)
		}

	case "http", "https":
		// http proxy
		l, err := sshListen(sshClientConn, remoteAddr)
		if err != nil {
			return nil, err
		}
		go Http(l)
	}

	// --------

	go func() {
		for {
			// Setup localConn (type net.Conn)
			localConn, err := localListener.Accept()
			if err != nil {
				break
			}

			go func() {
				// Setup sshConn (type net.Conn)
				sshConn, err := sshClientConn.Dial("tcp", remoteAddr)
				if err != nil {
					console.Error("Dial RemoteAddr:", err)
					return
				}

				// Copy localConn.Reader to sshConn.Writer
				go func() {
					_, _ = io.Copy(sshConn, localConn)
					_ = sshConn.Close()
					_ = localConn.Close()
				}()

				// Copy sshConn.Reader to localConn.Writer
				go func() {
					_, _ = io.Copy(localConn, sshConn)
					_ = sshConn.Close()
					_ = localConn.Close()
				}()

			}()
		}
	}()

	return stopChan, nil
}

func RemoteForward(username, password, serverAddr, remoteAddr, localAddr string, args ...string) (chan struct{}, error) {
	// Setup SSH config (type *ssh.ClientConfig)
	config := ssh.ClientConfig{
		User:    username,
		Auth:    []ssh.AuthMethod{ssh.Password(password)},
		Timeout: time.Second * 3,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// Setup sshClientConn (type *ssh.ClientConn)
	sshClientConn, err := ssh.Dial("tcp", serverAddr, &config)
	if err != nil {
		return nil, err
	}

	// create session
	session, err := sshClientConn.NewSession()
	if err != nil {
		return nil, err
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	remoteListener, err := sshListen(sshClientConn, remoteAddr)
	if err != nil {
		return nil, err
	}

	console.Info("RemoteForward", remoteAddr, "to", localAddr)

	var closeFn = func() {
		_ = remoteListener.Close()
		_ = sshClientConn.Close()
		_ = session.Close()
		console.Info("Close RemoteForward")
	}

	var stopChan = make(chan struct{}, 1)

	go func() {
		select {
		case <-stopChan:
			closeFn()
		}
	}()

	var ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				_, err := session.SendRequest(config.User, false, nil)
				if err != nil {
					closeFn()
					return
				}
			}
		}
	}()

	// -----------

	var proxyMode = tools.GetArgs([]string{"proxy", "--proxy"}, args)
	switch proxyMode {
	case "socks5":
		// socks5 proxy
		l, err := net.Listen("tcp", localAddr)
		if err != nil {
			return nil, err
		}
		go Socks5(l)

	case "tcp":
		// tcp proxy
		var target = tools.GetArgs([]string{proxyMode}, args)
		if target != "" {
			l, err := net.Listen("tcp", localAddr)
			if err != nil {
				return nil, err
			}
			go Tcp(l, target)
		}

	case "http", "https":
		// https proxy
		l, err := net.Listen("tcp", localAddr)
		if err != nil {
			return nil, err
		}
		go Http(l)
	}

	// -----------

	go func() {

		for {
			// Setup localConn (type net.Conn)
			remoteConn, err := remoteListener.Accept()
			if err != nil {
				break
			}

			go func() {

				// Setup localListener (type net.Listener)
				localConn, err := net.Dial("tcp", localAddr)
				if err != nil {
					console.Error("Dial localAddr:", err)
					return
				}

				// Copy localConn.Reader to sshConn.Writer
				go func() {
					_, _ = io.Copy(localConn, remoteConn)
					_ = localConn.Close()
					_ = remoteConn.Close()
				}()

				// Copy sshConn.Reader to localConn.Writer
				go func() {
					_, _ = io.Copy(remoteConn, localConn)
					_ = localConn.Close()
					_ = remoteConn.Close()
				}()

			}()
		}
	}()

	return stopChan, nil
}
