/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-10 00:29
**/

package ipc

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"

	jsoniter "github.com/json-iterator/go"
	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
)

var CallBack func([]string) string

var index int32 = 0

func handler() {
	var serverPipe = filepath.Join(app.Config.HomePath, "k8s-forward.server.pipe")
	server, err := os.OpenFile(serverPipe, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		console.Exit(err)
	}

	defer func() { _ = server.Close() }()

	var clientPipe = filepath.Join(app.Config.HomePath, "k8s-forward.client.pipe")
	client, err := os.OpenFile(clientPipe, os.O_RDWR, 0777)
	if err != nil {
		console.Exit(err)
	}

	defer func() { _ = client.Close() }()

	var buf = make([]byte, 1024)
	for {

		n, err := server.Read(buf)
		if err != nil {
			console.Exit(err)
		}

		func() {
			var i = atomic.AddInt32(&index, 1)
			defer atomic.AddInt32(&index, -1)
			if i != 1 {
				_, _ = client.WriteString("please wait command finished")
				return
			}

			var args []string
			err = jsoniter.Unmarshal(buf[:n], &args)
			if err != nil {
				console.Exit(err)
			}

			var res = CallBack(args)

			_, _ = client.WriteString(res)
		}()
	}
}

func Open() {
	var serverPipe = filepath.Join(app.Config.HomePath, "k8s-forward.server.pipe")
	err := syscall.Mkfifo(serverPipe, 0666)
	if err != nil {
		console.Exit(err)
	}

	var clientPipe = filepath.Join(app.Config.HomePath, "k8s-forward.client.pipe")
	err = syscall.Mkfifo(clientPipe, 0666)
	if err != nil {
		console.Exit(err)
	}

	go handler()
}

func Close() {
	var serverPipe = filepath.Join(app.Config.HomePath, "k8s-forward.server.pipe")
	var clientPipe = filepath.Join(app.Config.HomePath, "k8s-forward.client.pipe")
	_ = os.Remove(serverPipe)
	_ = os.Remove(clientPipe)
}

func Write(args []string) {

	var serverPipe = filepath.Join(app.Config.HomePath, "k8s-forward.server.pipe")
	server, err := os.OpenFile(serverPipe, os.O_RDWR, 0777)
	if err != nil {
		console.Exit("please run `k8s-forward connect` first")
	}

	defer func() { _ = server.Close() }()

	bts, err := jsoniter.Marshal(args)
	if err != nil {
		console.Exit(err)
	}

	_, err = server.Write(bts)
	if err != nil {
		console.Exit(err)
	}

	var clientPipe = filepath.Join(app.Config.HomePath, "k8s-forward.client.pipe")
	client, err := os.OpenFile(clientPipe, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		console.Exit(err)
	}

	defer func() { _ = client.Close() }()

	var buf = make([]byte, 1024)

	n, err := client.Read(buf)
	if err != nil {
		console.Exit(err)
	}

	console.Info(string(buf[:n]))
}
