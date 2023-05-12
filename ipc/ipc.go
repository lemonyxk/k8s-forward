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
	"bytes"
	"io"
	"net/http"
	"sync/atomic"

	jsoniter "github.com/json-iterator/go"
	"github.com/lemonyxk/console"
)

var CallBack func([]string) string

var index int32 = 0

func handler(writer http.ResponseWriter, request *http.Request) {
	var i = atomic.AddInt32(&index, 1)
	defer atomic.AddInt32(&index, -1)
	if i != 1 {
		_, _ = writer.Write([]byte("please wait command finished"))
		return
	}

	var bts, err = io.ReadAll(request.Body)
	if err != nil {
		console.Exit(err)
	}

	var args []string
	if err = jsoniter.Unmarshal(bts, &args); err != nil {
		console.Exit(err)
	}

	var res = CallBack(args)

	_, _ = writer.Write([]byte(res))
}

func Open() {
	go func() {
		http.HandleFunc("/", handler)
		err := http.ListenAndServe("0.0.0.0:29292", nil)
		if err != nil {
			console.Exit(err)
		}
	}()
}

func Close() {}

func Write(args []string) {
	bts, err := jsoniter.Marshal(args)
	if err != nil {
		console.Exit(err)
	}

	res, err := http.Post("http://127.0.0.1:29292", "application/json", bytes.NewReader(bts))
	if err != nil {
		console.Exit("please run `k8s-forward connect` first")
	}

	defer func() { _ = res.Body.Close() }()

	bts, err = io.ReadAll(res.Body)
	if err != nil {
		console.Exit(err)
	}

	console.Info(string(bts))
}
