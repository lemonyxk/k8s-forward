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
	"time"

	"github.com/lemonyxk/k8s-forward/ssh"
	"github.com/lemonyxk/k8s-forward/tools"
	"github.com/lemoyxk/console"
	"golang.org/x/term"
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
		var bts, err = term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			console.Error(err)
			return
		}
		password = string(bts)
	}

	var mode string
	if tools.HasArgs("-R", args) {
		mode = "-R"
	}
	if tools.HasArgs("-L", args) {
		mode = "-L"
	}

	var arr = strings.Split(server, "@")
	if len(arr) != 2 {
		console.Error("server addr is invalid")
		return
	}

	var user = arr[0]
	var serverAddr = arr[1]

	console.Info("user:", user, "server:", serverAddr, "remote:", remote, "local", local)

	// var reconnectInterval time.Duration = 0
	// var reconnect = tools.HasArgs("--reconnect", args)
	// if reconnect {
	// 	reconnectInterval = time.Second
	// }

	var config = ssh.Config{
		UserName:      user,
		Password:      password,
		ServerAddress: serverAddr,
		RemoteAddress: remote,
		LocalAddress:  local,
		Timeout:       time.Second * 3,
		// Reconnect:         reconnectInterval,
		HeartbeatInterval: time.Second * 1,
	}

	switch mode {
	case "-R":
		_, done, err := ssh.RemoteForward(config, args...)
		if err != nil {
			console.Error(err)
			return
		}

		select {
		case <-done:
			console.Info("remote forward done")
		}
	case "-L":
		_, done, err := ssh.LocalForward(config, args...)
		if err != nil {
			console.Error(err)
			return
		}

		select {
		case <-done:
			console.Info("local forward done")
		}
	default:
		console.Error("mode is required: -R or -L")
	}
}
