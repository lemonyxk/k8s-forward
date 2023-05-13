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

	"github.com/lemonyxk/k8s-forward/ipc"
	"github.com/lemonyxk/k8s-forward/k8s"
)

func Cmd() {
	switch os.Args[1] {
	case "connect":
		Clean(k8s.GetRecordFromFile())
		Connect()
	case "clean":
		Clean(k8s.GetRecordFromFile())
	case "help":
		println(Help())
	case "ssh":
		SSH()
	default:
		ipc.Write(os.Args)
	}
}
