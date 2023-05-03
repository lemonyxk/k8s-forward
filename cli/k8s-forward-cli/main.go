/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-06 01:38
**/
package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/exception"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/cmd"
	"github.com/lemonyxk/k8s-forward/config"
	"k8s.io/client-go/util/homedir"
)

func main() {

	console.DefaultLogger.InfoColor = console.FgGreen

	var home = homedir.HomeDir()

	exception.Assert.True(home != "")

	var kubePath = flag.String("kube", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kube config file")
	var recordPath = flag.String("record", filepath.Join(home, ".k8s-forward", "record.json"), "(optional) absolute path to the kube record file")

	flag.Parse()

	app.Config = &config.Config{KubePath: *kubePath, RecordPath: *recordPath}

	if len(os.Args) < 2 {
		console.Info(cmd.Help())
		return
	}

	cmd.Cmd(os.Args)

}
