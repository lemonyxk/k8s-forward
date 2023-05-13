/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-11 21:55
**/

package cmd

import (
	"os"
	"strings"
	"time"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/ssh"
	"github.com/lemonyxk/k8s-forward/utils"
	"golang.org/x/term"
)

func SSH() {

	var local = utils.GetArgs("-l", "--local")
	if local == "" {
		console.Error("local addr is required")
		return
	}

	var remote = utils.GetArgs("-r", "--remote")
	if remote == "" {
		console.Error("remote addr is required")
		return
	}

	var server = utils.GetArgs("-s", "--server")
	if server == "" {
		console.Error("server addr is required")
		return
	}

	var password = utils.GetArgs("-p", "--password")
	var hasPassword = utils.HasArgs("-p", "--password")
	var privateKey = utils.GetArgs("-k", "--key")
	if password == "" && hasPassword && privateKey == "" {
		console.Infof("password: ")
		var bts, err = term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			console.Error(err)
			return
		}
		password = string(bts)
	}

	var mode string
	if utils.HasArgs("-R") {
		mode = "-R"
	}
	if utils.HasArgs("-L") {
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

	var config = ssh.ForwardConfig{
		UserName:          user,
		Password:          password,
		PrivateKey:        privateKey,
		ServerAddress:     serverAddr,
		RemoteAddress:     remote,
		LocalAddress:      local,
		Timeout:           time.Second * 3,
		HeartbeatInterval: time.Second * 1,
	}

	switch mode {
	case "-R":
		_, done, err := ssh.RemoteForward(config)
		if err != nil {
			console.Error(err)
			return
		}

		select {
		case <-done:
			console.Info("remote forward done")
		}
	case "-L":
		_, done, err := ssh.LocalForward(config)
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
