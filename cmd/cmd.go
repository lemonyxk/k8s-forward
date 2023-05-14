/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-11 23:15
**/

package cmd

import (
	"os"

	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/ipc"
)

func Cmd() {
	switch os.Args[1] {
	case "connect":
		Connect()
	case "clean":
		Clean(app.LoadAllServices())
	case "help":
		println(Help())
	case "ssh":
		SSH()
	default:
		ipc.Write(os.Args)
	}
}
