/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-11 23:15
**/

package cmd

import (
	"os"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/ipc"
	"github.com/lemonyxk/k8s-forward/k8s"
)

func Cmd() {

	console.DefaultLogger.Flags = console.NONE

	switch os.Args[1] {
	case "connect":
		console.DefaultLogger.Flags = console.TIME | console.FILE
		console.DefaultLogger.InfoColor = console.FgGreen
		console.DefaultLogger.Colorful = true
		Clean(k8s.GetRecordFromFile())
		Connect()
	case "clean":
		Clean(k8s.GetRecordFromFile())
	case "help":
		console.Info(Help())
	case "ssh":
		SSH()
	default:
		ipc.Write(os.Args)
	}
}
