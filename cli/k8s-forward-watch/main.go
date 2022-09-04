/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-09-04 14:40
**/

package main

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/lemonyxk/k8s-forward/tools"
	"github.com/lemoyxk/console"
	"github.com/lemoyxk/utils"
)

func main() {

	console.SetFlags(console.TIME)
	console.SetInfoColor(console.FgGreen)

	var debug = tools.HasArgs("--debug", os.Args)
	if debug {
		console.SetFlags(console.FILE | console.TIME)
	}

	var reconnect = tools.HasArgs("--reconnect", os.Args)

	var reconnectTimes = -1

	for {

		reconnectTimes++

		if reconnectTimes > 0 {
			console.Error("Reconnecting", reconnectTimes, "times")
		}

		var err = Run()
		if err != nil {
			console.Error(err)
		}

		if !reconnect {
			return
		}

		time.Sleep(time.Second)
	}
}

func Run() error {
	var commands []string
	commands = append(commands, os.Args[1:]...)
	// commands = append(commands, "--debug")

	var cmd = exec.Command("k8s-forward", commands...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	var err = cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}

func Kill(pid int) {
	var err = utils.Signal.KillGroup(pid, syscall.SIGKILL)
	if err != nil {
		console.Error(err)
	}
}
