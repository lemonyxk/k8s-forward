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

type Config struct {
	UserName          string
	Password          string
	ServerAddress     string
	RemoteAddress     string
	LocalAddress      string
	Timeout           time.Duration
	Reconnect         time.Duration
	HeartbeatInterval time.Duration
}

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
			_ = conn.Close()
			time.Sleep(time.Second)
			return sshListen(sshClientConn, remoteAddr)
		}
		return nil, err
	}

	return l, nil
}

func LocalForward(cfg Config, args ...string) (chan struct{}, chan struct{}, error) {

	var stopChan = make(chan struct{}, 1)
	var doneChan = make(chan struct{}, 1)

	var fn func() error

	fn = func() error {

		var stop = make(chan struct{}, 1)
		var isStop = false
		var done = make(chan struct{}, 1)

		// Setup SSH config (type *ssh.ClientConfig)
		var config = &ssh.ClientConfig{
			User:    cfg.UserName,
			Auth:    []ssh.AuthMethod{ssh.Password(cfg.Password)},
			Timeout: cfg.Timeout,
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}

		// Setup sshClientConn (type *ssh.ClientConn)

		var sshClientConn *ssh.Client
		var err error
		var l net.Listener

		for {
			sshClientConn, err = ssh.Dial("tcp", cfg.ServerAddress, config)
			if err == nil {
				break
			}

			if cfg.Reconnect == 0 {
				doneChan <- struct{}{}
				return err
			}

			console.Error(err)
			console.Info("Reconnecting...")
			time.Sleep(cfg.Reconnect)
		}

		// create session
		session, err := sshClientConn.NewSession()
		if err != nil {
			return err
		}

		session.Stdout = os.Stdout
		session.Stderr = os.Stderr
		session.Stdin = os.Stdin

		// Setup localListener (type net.Listener)
		localListener, err := net.Listen("tcp", cfg.LocalAddress)
		if err != nil {
			return err
		}

		console.Info("LocalForward", cfg.LocalAddress, "to", cfg.RemoteAddress)

		var closeFn = func() {
			if isStop {
				return
			}
			isStop = true
			_ = localListener.Close()
			_ = sshClientConn.Close()
			_ = session.Close()
			if l != nil {
				_ = l.Close()
			}
			console.Info("Close LocalForward")
		}

		go func() {
			for {
				select {
				case <-stop:
					cfg.Reconnect = 0
					closeFn()
					return
				case <-stopChan:
					stop <- struct{}{}
				case <-done:
					return
				}
			}
		}()

		go func() {
			var t = time.NewTimer(cfg.Timeout)

			for {
				time.Sleep(cfg.HeartbeatInterval)

				var ch = make(chan struct{})
				go func() {
					_, err = session.SendRequest(config.User, true, nil)
					if err == nil {
						ch <- struct{}{}
					}
				}()
				select {
				case <-t.C:
					closeFn()

					if cfg.Reconnect == 0 {
						doneChan <- struct{}{}
						return
					} else {
						done <- struct{}{}
					}

					console.Info("Reconnecting...")
					time.Sleep(cfg.Reconnect)
					var err = fn()
					if err != nil {
						console.Error(err)
					}

					return
				case <-ch:
					t.Reset(cfg.Timeout)
				}
			}
		}()

		// --------

		var proxyMode = tools.GetArgs([]string{"proxy", "--proxy"}, args)
		switch proxyMode {
		case "socks5":
			// socks5 proxy
			l, err = sshListen(sshClientConn, cfg.RemoteAddress)
			if err != nil {
				return err
			}
			go Socks5(l)

		case "tcp":
			// tcp proxy
			var target = tools.GetArgs([]string{proxyMode}, args)
			if target != "" {
				l, err = sshListen(sshClientConn, cfg.RemoteAddress)
				if err != nil {
					return err
				}
				go Tcp(l, target)
			}

		case "http", "https":
			// http proxy
			l, err = sshListen(sshClientConn, cfg.RemoteAddress)
			if err != nil {
				return err
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
					sshConn, err := sshClientConn.Dial("tcp", cfg.RemoteAddress)
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

		return nil
	}

	var err = fn()

	return stopChan, doneChan, err
}

func RemoteForward(cfg Config, args ...string) (chan struct{}, chan struct{}, error) {

	var stopChan = make(chan struct{}, 1)
	var doneChan = make(chan struct{}, 1)

	var fn func() error

	fn = func() error {

		var stop = make(chan struct{}, 1)
		var isStop = false
		var done = make(chan struct{}, 1)

		// Setup SSH config (type *ssh.ClientConfig)
		var config = ssh.ClientConfig{
			User:    cfg.UserName,
			Auth:    []ssh.AuthMethod{ssh.Password(cfg.Password)},
			Timeout: cfg.Timeout,
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}

		// Setup sshClientConn (type *ssh.ClientConn)

		var sshClientConn *ssh.Client
		var err error
		var l net.Listener

		for {
			sshClientConn, err = ssh.Dial("tcp", cfg.ServerAddress, &config)
			if err == nil {
				break
			}

			if cfg.Reconnect == 0 {
				doneChan <- struct{}{}
				return err
			}

			console.Error(err)
			console.Info("Reconnecting...")
			time.Sleep(cfg.Reconnect)
		}

		// create session
		session, err := sshClientConn.NewSession()
		if err != nil {
			return err
		}

		session.Stdout = os.Stdout
		session.Stderr = os.Stderr
		session.Stdin = os.Stdin

		remoteListener, err := sshListen(sshClientConn, cfg.RemoteAddress)
		if err != nil {
			return err
		}

		console.Info("RemoteForward", cfg.RemoteAddress, "to", cfg.LocalAddress)

		var closeFn = func() {
			if isStop {
				return
			}
			isStop = true
			// will block when net broken
			go func() { _ = remoteListener.Close() }()
			_ = sshClientConn.Close()
			_ = session.Close()
			if l != nil {
				_ = l.Close()
			}
			console.Info("Close RemoteForward")
		}

		go func() {
			for {
				select {
				case <-stop:
					cfg.Reconnect = 0
					closeFn()
					return
				case <-stopChan:
					stop <- struct{}{}
				case <-done:
					return
				}
			}
		}()

		go func() {
			var t = time.NewTimer(cfg.Timeout)

			for {
				time.Sleep(cfg.HeartbeatInterval)

				var ch = make(chan struct{})
				go func() {
					_, err = session.SendRequest(config.User, true, nil)
					if err == nil {
						ch <- struct{}{}
					}
				}()
				select {
				case <-t.C:
					closeFn()

					if cfg.Reconnect == 0 {
						doneChan <- struct{}{}
						return
					} else {
						done <- struct{}{}
					}

					console.Info("Reconnecting...")
					time.Sleep(cfg.Reconnect)
					var err = fn()
					if err != nil {
						console.Error(err)
					}

					return
				case <-ch:
					t.Reset(cfg.Timeout)
				}
			}
		}()

		// -----------

		var proxyMode = tools.GetArgs([]string{"proxy", "--proxy"}, args)
		switch proxyMode {
		case "socks5":
			// socks5 proxy
			l, err = net.Listen("tcp", cfg.LocalAddress)
			if err != nil {
				return err
			}
			go Socks5(l)

		case "tcp":
			// tcp proxy
			var target = tools.GetArgs([]string{proxyMode}, args)
			if target != "" {
				l, err = net.Listen("tcp", cfg.LocalAddress)
				if err != nil {
					return err
				}
				go Tcp(l, target)
			}

		case "http", "https":
			// https proxy
			l, err = net.Listen("tcp", cfg.LocalAddress)
			if err != nil {
				return err
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
					localConn, err := net.Dial("tcp", cfg.LocalAddress)
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

		return nil
	}

	var err = fn()

	return stopChan, doneChan, err
}
