/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-09-04 14:40
**/

package main

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/utils"
	utils2 "github.com/lemoyxk/utils"
)

func main() {

	console.DefaultLogger.Flags = console.TIME | console.FILE
	console.DefaultLogger.InfoColor = console.FgGreen
	console.DefaultLogger.Colorful = true

	var reconnect = utils.HasArgs("--reconnect")

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

	var cmd = exec.Command("k8s-forward-cli", commands...)
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
	var err = utils2.Signal.KillGroup(pid, syscall.SIGKILL)
	if err != nil {
		console.Error(err)
	}
}
