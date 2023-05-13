/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-09 16:28
**/

package cmd

import (
	"os"

	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemonyxk/k8s-forward/dns"
	"github.com/lemonyxk/k8s-forward/ipc"
	"github.com/lemonyxk/k8s-forward/k8s"
	"github.com/lemonyxk/k8s-forward/net"
)

func Clean(record *config.Record) {

	if record == nil {
		return
	}

	if app.RestConfig == nil {
		app.RestConfig = k8s.NewRestConfig()
	}

	if app.Client == nil {
		app.Client = k8s.NewClient()
	}

	// k8s.UnForwardServiceAll()

	ipc.Close()

	net.DeleteNetWork(record)

	dns.DeleteNameServer(record)

	UnScaleAll(record)

	UnDeploymentAll(record)

	_ = os.RemoveAll(app.Config.RecordPath)
}
