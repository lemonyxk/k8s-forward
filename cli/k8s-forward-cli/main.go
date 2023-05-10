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
	"os"
	"path/filepath"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/exception"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/cmd"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemonyxk/k8s-forward/tools"
	"k8s.io/client-go/util/homedir"
)

func main() {

	console.DefaultLogger.InfoColor = console.FgGreen

	var home = homedir.HomeDir()

	exception.Assert.True(home != "")

	var kubePath = tools.GetArgs([]string{"-kube", "--kubeconfig"}, os.Args)
	var recordPath = tools.GetArgs([]string{"-record", "--record"}, os.Args)

	if kubePath == "" {
		kubePath = filepath.Join(home, ".kube", "config")
	}

	if recordPath == "" {
		recordPath = filepath.Join(home, ".k8s-forward", "record.json")
	}

	app.Config = &config.Config{KubePath: kubePath, RecordPath: recordPath}

	if len(os.Args) < 2 {
		console.Info(cmd.Help())
		return
	}

	cmd.Cmd(os.Args)

}
