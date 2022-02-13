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
	"strings"
	"time"

	"github.com/lemoyxk/console"
	"github.com/lemoyxk/k8s-forward/tools"
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
	console.Info(serverAddr)
	sshClientConn, err := ssh.Dial("tcp", serverAddr, config)
	if err != nil {
		return nil, err
	}

	// Setup localListener (type net.Listener)
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return nil, err
	}

	var stopChan = make(chan struct{}, 1)

	go func() {
		select {
		case <-stopChan:
			_ = localListener.Close()
			_ = sshClientConn.Close()
		}
	}()

	// --------

	// http proxy
	var list []string
	var http = tools.GetArgs([]string{"http", "--http", "https", "--https"}, args)
	if http != "" {
		list = append(list, http)
		var to = tools.GetArgs([]string{http}, args)
		if to != "" {
			list = append(list, to)
		}
	}

	if len(list) > 0 {
		l, err := sshClientConn.Listen("tcp", remoteAddr)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(http, "http") {
			go Http(l, "http", list)
		}
		if strings.HasSuffix(http, "https") {
			go Http(l, "https", list)
		}
	}

	// tcp proxy
	var tcp = tools.GetArgs([]string{"tcp", "--tcp"}, args)
	if tcp != "" {
		l, err := sshClientConn.Listen("tcp", remoteAddr)
		if err != nil {
			return nil, err
		}
		go Tcp(l, tcp)
	}

	// socks5 proxy
	var socks5 = tools.HasArgs("-S", args)
	if socks5 {
		l, err := sshClientConn.Listen("tcp", remoteAddr)
		if err != nil {
			return nil, err
		}
		go Socks5(l)
	}

	// --------

	go func() {
		for {
			// Setup localConn (type net.Conn)
			localConn, err := localListener.Accept()
			if err != nil {
				return
			}

			console.Info("localConn:", localConn.RemoteAddr().String())

			go func() {
				// Setup sshConn (type net.Conn)
				sshConn, err := sshClientConn.Dial("tcp", remoteAddr)
				if err != nil {
					console.Error(err)
					return
				}

				// Copy localConn.Reader to sshConn.Writer
				go func() {
					_, err = io.Copy(sshConn, localConn)
					if err != nil {
						console.Error(err)
					}
					_ = sshConn.Close()
					_ = localConn.Close()
				}()

				// Copy sshConn.Reader to localConn.Writer
				go func() {
					_, err = io.Copy(localConn, sshConn)
					if err != nil {
						console.Error(err)
					}
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

	remoteListener, err := sshClientConn.Listen("tcp", remoteAddr)
	if err != nil {
		return nil, err
	}

	var stopChan = make(chan struct{}, 1)

	go func() {
		select {
		case <-stopChan:
			_ = remoteListener.Close()
			_ = sshClientConn.Close()
		}
	}()

	// -----------

	// http proxy
	var list []string
	var http = tools.GetArgs([]string{"http", "--http", "https", "--https"}, args)
	if http != "" {
		list = append(list, http)
		var to = tools.GetArgs([]string{http}, args)
		if to != "" {
			list = append(list, to)
		}
	}

	if len(list) > 0 {
		l, err := net.Listen("tcp", localAddr)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(http, "http") {
			go Http(l, "http", list)
		}
		if strings.HasSuffix(http, "https") {
			go Http(l, "https", list)
		}
	}

	// tcp proxy
	var tcp = tools.GetArgs([]string{"tcp", "--tcp"}, args)
	if tcp != "" {
		l, err := net.Listen("tcp", localAddr)
		if err != nil {
			return nil, err
		}
		go Tcp(l, tcp)
	}

	// socks5 proxy
	var socks5 = tools.HasArgs("-S", args)
	if socks5 {
		l, err := net.Listen("tcp", localAddr)
		if err != nil {
			return nil, err
		}
		go Socks5(l)
	}

	// -----------

	go func() {

		for {
			// Setup localConn (type net.Conn)
			remoteConn, err := remoteListener.Accept()
			if err != nil {
				break
			}

			console.Info("remoteConn:", remoteConn.RemoteAddr().String())

			go func() {

				// Setup localListener (type net.Listener)
				localConn, err := net.Dial("tcp", localAddr)
				if err != nil {
					console.Error(err)
					return
				}

				// Copy localConn.Reader to sshConn.Writer
				go func() {
					_, err = io.Copy(localConn, remoteConn)
					if err != nil {
						console.Error(err)
					}
					_ = localConn.Close()
					_ = remoteConn.Close()
				}()

				// Copy sshConn.Reader to localConn.Writer
				go func() {
					_, err = io.Copy(remoteConn, localConn)
					if err != nil {
						console.Error(err)
					}
					_ = localConn.Close()
					_ = remoteConn.Close()
				}()

			}()
		}
	}()

	return stopChan, nil
}
