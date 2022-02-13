/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-11 21:55
**/

package cmd

import (
	"os"
	"strings"

	"github.com/lemoyxk/console"
	"github.com/lemoyxk/k8s-forward/ssh"
	"github.com/lemoyxk/k8s-forward/tools"
	"golang.org/x/crypto/ssh/terminal"
)

func SSH(args []string) {

	var local = tools.GetArgs([]string{"local", "-l", "--local"}, args)
	if local == "" {
		console.Error("local addr is required")
		return
	}

	var remote = tools.GetArgs([]string{"remote", "-r", "--remote"}, args)
	if remote == "" {
		console.Error("remote addr is required")
		return
	}

	var server = tools.GetArgs([]string{"server", "-s", "--server"}, args)
	if server == "" {
		console.Error("server addr is required")
		return
	}

	var password = tools.GetArgs([]string{"password", "-p", "--password"}, args)
	if password == "" {
		console.Infof("Password: ")
		var bts, err = terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			console.Error(err)
			return
		}
		password = string(bts)
	}

	var r = tools.HasArgs("-R", args)
	var l = tools.HasArgs("-L", args)

	var arr = strings.Split(server, "@")
	if len(arr) != 2 {
		console.Error("server addr is invalid")
		return
	}

	var user = arr[0]
	var serverAddr = arr[1]

	console.Info("user:", user, "server:", serverAddr, "remote:", remote, "local", local)

	if r {

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
			if strings.HasSuffix(http, "http") {
				go ssh.Http("http", local, list)
			}
			if strings.HasSuffix(http, "https") {
				go ssh.Http("https", local, list)
			}
		}

		// tcp proxy
		var tcp = tools.GetArgs([]string{"tcp", "--tcp"}, args)
		if tcp != "" {
			go ssh.Tcp(local, tcp)
		}

		// socks5 proxy
		var socks5 = tools.HasArgs("-S", args)
		if socks5 {
			go ssh.Socks5(local)
		}

		st, err := ssh.RemoteForward(user, password, serverAddr, remote, local)
		if err != nil {
			console.Error(err)
			return
		}
		select {
		case <-st:
			console.Info("remote forward done")
		}
	}

	if l {
		st, err := ssh.LocalForward(user, password, serverAddr, remote, local)
		if err != nil {
			console.Error(err)
			return
		}
		select {
		case <-st:
			console.Info("local forward done")
		}
	}

	console.Error("mode is required: -R or -L")
}
