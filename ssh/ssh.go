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
	"path/filepath"
	"strings"
	"time"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/tools"
	"golang.org/x/crypto/ssh"
)

type ForwardConfig struct {
	UserName          string
	Password          string
	PrivateKey        string
	ServerAddress     string
	RemoteAddress     string
	LocalAddress      string
	Timeout           time.Duration
	HeartbeatInterval time.Duration
}

type Config struct {
	UserName   string
	Password   string
	Addr       string
	PrivateKey string
	Timeout    time.Duration
}

func SSH(config Config) (*ssh.Client, error) {
	var (
		auth         []ssh.AuthMethod
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		err          error
	)

	if config.Password == "" {

		var pemBytes []byte

		if config.PrivateKey == "" {
			// read private key file
			var homeDir, err = os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("get home dir failed %v", err)
			}
			var privateKeyPath = homeDir + "/.ssh/id_rsa"
			pemBytes, err = os.ReadFile(privateKeyPath)
			if err != nil {
				return nil, fmt.Errorf("reading private key file failed %v", err)
			}
		} else {
			var absPath, err = filepath.Abs(config.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("get abs path failed %v", err)
			}
			pemBytes, err = os.ReadFile(absPath)
			if err != nil {
				return nil, fmt.Errorf("reading private key file failed %v", err)
			}
		}

		// create signer
		// generate signer instance from plain key
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing plain private key failed %v", err)
		}

		auth = append(auth, ssh.PublicKeys(signer))
	} else {
		// get auth method
		auth = append(auth, ssh.Password(config.Password))
	}

	clientConfig = &ssh.ClientConfig{
		User:    config.UserName,
		Auth:    auth,
		Timeout: config.Timeout,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	if client, err = ssh.Dial("tcp", config.Addr, clientConfig); err != nil {
		return nil, err
	}

	return client, nil
}

func sshListen(sshClientConn *ssh.Client, remoteAddr string) (net.Listener, error) {
	l, err := sshClientConn.Listen("tcp", remoteAddr)
	if err != nil {
		// net broken, not closed
		if strings.HasSuffix(err.Error(), "tcp forward request denied by peer") {
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

func LocalForward(cfg ForwardConfig) (chan struct{}, chan struct{}, error) {

	var stopChan = make(chan struct{}, 1)
	var doneChan = make(chan struct{}, 1)

	var stop = make(chan struct{}, 1)
	var isStop = false
	var isClose = false

	// Setup sshClientConn (type *ssh.ClientConn)

	var sshClientConn *ssh.Client
	var session *ssh.Session
	var localListener net.Listener
	var err error
	var l net.Listener

	sshClientConn, err = SSH(Config{UserName: cfg.UserName, Password: cfg.Password, Addr: cfg.ServerAddress, Timeout: cfg.Timeout})
	if err != nil {
		console.Exit(err)
	}

	// create session
	session, err = sshClientConn.NewSession()
	if err != nil {
		console.Exit(err)
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// Setup localListener (type net.Listener)
	localListener, err = net.Listen("tcp", cfg.LocalAddress)
	if err != nil {
		console.Exit(err)
	}

	console.Info("local forward", cfg.LocalAddress, "to", cfg.RemoteAddress)

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

		console.Info("close local forward")
	}

	go func() {
		for {
			select {
			case <-stop:
				closeFn()
				doneChan <- struct{}{}
				return
			case <-stopChan:
				isClose = true
				stop <- struct{}{}
			}
		}
	}()

	go func() {
		var t = time.NewTimer(cfg.Timeout)

		for {
			time.Sleep(cfg.HeartbeatInterval)

			var ch = make(chan struct{})
			go func() {
				_, err = session.SendRequest(cfg.UserName, true, nil)
				if err == nil {
					ch <- struct{}{}
				}
			}()
			select {
			case <-t.C:
				if !isClose {
					closeFn()
					console.Exit(err)
				}
			case <-ch:
				t.Reset(cfg.Timeout)
			}
		}
	}()

	// --------

	var proxyMode = tools.GetArgs("--proxy")
	switch proxyMode {
	case "socks5":
		// socks5 proxy
		l, err = sshListen(sshClientConn, cfg.RemoteAddress)
		if err != nil {
			console.Exit(err)
		}
		go func() {
			var err = Socks5(l)
			if !isClose {
				console.Exit(err)
			} else {
				console.Error(err)
			}
		}()

	case "tcp":
		// tcp proxy
		var target = tools.GetArgs("--target")
		if target == "" {
			console.Exit("target argument is required")
		}

		l, err = sshListen(sshClientConn, cfg.RemoteAddress)
		if err != nil {
			console.Exit(err)
		}
		go func() {
			var err = Tcp(l, target)
			if !isClose {
				console.Exit(err)
			} else {
				console.Error(err)
			}
		}()

	case "http", "https":
		// http proxy
		l, err = sshListen(sshClientConn, cfg.RemoteAddress)
		if err != nil {
			console.Exit(err)
		}
		go func() {
			var err = Http(l)
			if !isClose {
				console.Exit(err)
			} else {
				console.Error(err)
			}
		}()
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
					console.Error("dial remote addr:", cfg.RemoteAddress, err)
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

		if !isClose {
			console.Exit("localListener closed")
		} else {
			console.Error("localListener closed")
		}
	}()

	return stopChan, doneChan, err
}

func RemoteForward(cfg ForwardConfig) (chan struct{}, chan struct{}, error) {

	var stopChan = make(chan struct{}, 1)
	var doneChan = make(chan struct{}, 1)

	var stop = make(chan struct{}, 1)
	var isStop = false
	var isClose = false

	// Setup sshClientConn (type *ssh.ClientConn)

	var sshClientConn *ssh.Client
	var session *ssh.Session
	var remoteListener net.Listener
	var err error
	var l net.Listener

	sshClientConn, err = SSH(Config{UserName: cfg.UserName, Password: cfg.Password, Addr: cfg.ServerAddress, Timeout: cfg.Timeout})
	if err != nil {
		console.Exit(err)
	}

	// create session
	session, err = sshClientConn.NewSession()
	if err != nil {
		console.Exit(err)
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	remoteListener, err = sshListen(sshClientConn, cfg.RemoteAddress)
	if err != nil {
		console.Exit(err)
	}

	console.Info("remote forward", cfg.RemoteAddress, "to", cfg.LocalAddress)

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

		console.Info("close remote forward")
	}

	go func() {
		for {
			select {
			case <-stop:
				closeFn()
				doneChan <- struct{}{}
				return
			case <-stopChan:
				isClose = true
				stop <- struct{}{}
			}
		}
	}()

	go func() {
		var t = time.NewTimer(cfg.Timeout)

		for {
			time.Sleep(cfg.HeartbeatInterval)

			var ch = make(chan struct{})
			go func() {
				_, err = session.SendRequest(cfg.UserName, true, nil)
				if err == nil {
					ch <- struct{}{}
				}
			}()
			select {
			case <-t.C:
				if !isClose {
					closeFn()
					console.Exit(err)
				}
			case <-ch:
				t.Reset(cfg.Timeout)
			}
		}
	}()

	// -----------

	var proxyMode = tools.GetArgs("--proxy")
	switch proxyMode {
	case "socks5":
		// socks5 proxy
		l, err = net.Listen("tcp", cfg.LocalAddress)
		if err != nil {
			console.Exit(err)
		}
		go func() {
			var err = Socks5(l)
			if !isClose {
				console.Exit(err)
			} else {
				console.Error(err)
			}
		}()

	case "tcp":
		// tcp proxy
		var target = tools.GetArgs("--target")
		if target == "" {
			console.Exit("target argument is required")
		}

		l, err = net.Listen("tcp", cfg.LocalAddress)
		if err != nil {
			console.Exit(err)
		}
		go func() {
			var err = Tcp(l, target)
			if !isClose {
				console.Exit(err)
			} else {
				console.Error(err)
			}
		}()

	case "http", "https":
		// https proxy
		l, err = net.Listen("tcp", cfg.LocalAddress)
		if err != nil {
			console.Exit(err)
		}
		go func() {
			var err = Http(l)
			if !isClose {
				console.Exit(err)
			} else {
				console.Error(err)
			}
		}()
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
					console.Error("Dial localAddr:", cfg.LocalAddress, err)
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

		if !isClose {
			console.Exit("remoteListener closed")
		} else {
			console.Error("remoteListener closed")
		}
	}()

	return stopChan, doneChan, err
}
