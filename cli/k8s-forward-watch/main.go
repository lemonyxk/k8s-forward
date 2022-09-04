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
	"strings"
	"syscall"
	"time"

	"github.com/lemoyxk/console"
	"github.com/lemoyxk/utils"
)

func main() {
	for {
		var err = Run()
		if err != nil {
			console.Error(err)
		}
		time.Sleep(time.Second)
	}
}

func Run() error {
	var commands []string
	commands = append(commands, "k8s-forward")
	commands = append(commands, os.Args[1:]...)

	var cmd = utils.Cmd.New(strings.Join(commands, " ")).Cmd()
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
