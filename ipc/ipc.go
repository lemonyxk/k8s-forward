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
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/lemoyxk/console"
	"github.com/lemoyxk/k8s-forward/tools"
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
	_, _ = writer.Write([]byte(CallBack(strings.Split(request.URL.Path[1:], " "))))
}

func Open() {
	go func() {
		http.HandleFunc("/", handler)
		err := http.ListenAndServe("0.0.0.0:29292", nil)
		if err != nil {
			tools.Exit(err)
		}
	}()
}

func Close() {}

func Write(str string) {

	var res, err = http.Get("http://127.0.0.1:29292/" + str)
	if err != nil {
		tools.Exit("please run `k8s-forward connect` first")
	}
	defer func() { _ = res.Body.Close() }()

	bts, err := io.ReadAll(res.Body)
	if err != nil {
		tools.Exit(err)
	}

	console.Info(string(bts))
}
