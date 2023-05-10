/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-09 16:24
**/

package cmd

import (
	"os"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/dns"
	"github.com/lemonyxk/k8s-forward/ipc"
	"github.com/lemonyxk/k8s-forward/k8s"
	"github.com/lemonyxk/k8s-forward/net"
	"github.com/lemonyxk/k8s-forward/tools"
	"github.com/lemonyxk/promise"
	"github.com/lemoyxk/utils"
)

func Connect() {
	console.Infof("\r%s\n", "start k8s-forward...")

	var namespace = tools.GetArgs([]string{"--namespace", "-n"}, os.Args)
	if namespace == "" {
		namespace = "default"
	}

	app.RestConfig = k8s.NewRestConfig()

	app.Client = k8s.NewClient()

	app.Record = k8s.GetRecord(namespace)

	k8s.SaveRecordToFile(app.Record)

	app.DnsDomain = dns.GetDNSDomain()

	ipc.Open()

	ipc.CallBack = Default

	var p1 = promise.New(func(resolve func(int), reject func(error)) {
		dns.AddNameServer()
		net.CreateNetWork(app.Record)
		resolve(0)
	})

	var p2 = promise.New(func(resolve func(int), reject func(error)) {
		dns.StartDNS(func() {
			resolve(0)
		})
	})

	var p3 = promise.New(func(resolve func(int), reject func(error)) {
		// k8s.ForwardHost()
		resolve(0)
	})

	promise.Fall(p1, p2, p3).Then(func(result []int) {
		console.Info("k8s-forward up")
	})

	k8s.Render()

	utils.Signal.ListenKill().Done(func(sig os.Signal) {
		console.Warningf("\r%s\n", "close k8s-forward...")
		Clean(app.Record)
		console.Warning("k8s-forward down")
	})
}
