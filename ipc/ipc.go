/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
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
		println(err.Error())
		return
	}

	var args []string
	err = jsoniter.Unmarshal(bts, &args)
	if err != nil {
		println(err.Error())
		return
	}

	var res = CallBack(args)

	_, _ = writer.Write([]byte(res))
}

func Open() {
	go func() {
		http.HandleFunc("/", handler)
		err := http.ListenAndServe("0.0.0.0:29292", nil)
		if err != nil {
			println(err.Error())
		}
	}()
}

func Close() {}

func Write(args []string) {
	bts, err := jsoniter.Marshal(args)
	if err != nil {
		println(err.Error())
		return
	}

	res, err := http.Post("http://127.0.0.1:29292", "application/json", bytes.NewReader(bts))
	if err != nil {
		println("please run `k8s-forward connect` first")
		return
	}

	defer func() { _ = res.Body.Close() }()

	bts, err = io.ReadAll(res.Body)
	if err != nil {
		println(err.Error())
		return
	}

	println(string(bts))
}
