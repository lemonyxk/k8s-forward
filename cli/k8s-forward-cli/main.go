/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
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
	"github.com/lemonyxk/k8s-forward/utils"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/homedir"
)

func Init() {
	var log = console.NewLogger()
	log.Flags = console.TIME | console.FILE
	log.InfoColor = console.FgWhite
	log.ErrorColor = console.FgMagenta
	log.Colorful = true
	log.Deep = 6

	runtime.ErrorHandlers = []func(error){
		func(err error) {
			log.Error(err)
		},
	}

	console.DefaultLogger.Flags = console.TIME | console.FILE
	console.DefaultLogger.InfoColor = console.FgGreen
	console.DefaultLogger.Colorful = true
}

func main() {

	Init()

	var home = homedir.HomeDir()
	exception.Assert.True(home != "")

	var kubePath = utils.GetArgs("-kube", "--kubeconfig")
	var homePath = filepath.Join(home, ".k8s-forward")

	if kubePath == "" {
		kubePath = filepath.Join(home, ".kube", "config")
	}

	var err = os.MkdirAll(homePath, 0755)
	exception.Assert.True(err == nil)

	app.Config = &config.Config{KubePath: kubePath, HomePath: homePath}

	if len(os.Args) < 2 {
		console.Info(cmd.Help())
		return
	}

	cmd.Cmd()
}
