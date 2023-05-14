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
	"github.com/lemonyxk/k8s-forward/dns"
	"github.com/lemonyxk/k8s-forward/ipc"
	"github.com/lemonyxk/k8s-forward/k8s"
	"github.com/lemonyxk/k8s-forward/services"
)

func Clean(services *services.Services) {

	if services == nil {
		return
	}

	if len(services.Namespaces()) == 0 {
		return
	}

	if app.RestConfig == nil {
		app.RestConfig = k8s.NewRestConfig()
	}

	if app.Client == nil {
		app.Client = k8s.NewClient()
	}

	ipc.Close()

	app.DeleteNetWork(services)

	dns.DeleteNameServer(services)

	k8s.UnSwitchScaleAll(services)

	k8s.UnSwitchDeploymentAll(services)

	_ = os.RemoveAll(app.Config.HomePath)
}
