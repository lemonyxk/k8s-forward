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
	"strings"

	"github.com/lemonyxk/k8s-forward/ipc"
	"github.com/lemonyxk/k8s-forward/k8s"
	"github.com/lemoyxk/console"
)

func Cmd(args []string) {

	switch args[1] {
	case "connect":
		Clean(k8s.GetRecordFromFile())
		Connect()
	case "clean":
		Clean(k8s.GetRecordFromFile())
	case "help":
		console.Info(Help())
	case "ssh":
		SSH(args[1:])
	default:
		ipc.Write(strings.Join(args[1:], " "))
	}

}
