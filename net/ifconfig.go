/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-08 18:18
**/

package net

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/lemoyxk/console"
	"github.com/lemoyxk/k8s-forward/config"
	"github.com/lemoyxk/k8s-forward/tools"
)

func CreateNetWorkByIp(ip string) {
	if runtime.GOOS == "linux" {
		createLinux([]string{ip})
	} else if runtime.GOOS == "darwin" {
		createDarwin([]string{ip})
	} else {
		tools.Exit("not support windows")
	}
}

func CreateNetWork(record *config.Record) {
	var ip []string

	for i := 0; i < len(record.Services); i++ {

		if record.Services[i].SelectPod == nil {
			continue
		}

		if record.Services[i].Switch != nil {
			ip = append(ip, record.Services[i].Switch.Pod.IP)
		}

		ip = append(ip, record.Services[i].SelectPod.IP)
	}

	if runtime.GOOS == "linux" {
		createLinux(ip)
	} else if runtime.GOOS == "darwin" {
		createDarwin(ip)
	} else {
		tools.Exit("not support windows")
	}
}

func DeleteNetWorkByIp(ip string) {
	if runtime.GOOS == "linux" {
		deleteLinux([]string{ip})
	} else if runtime.GOOS == "darwin" {
		deleteDarwin([]string{ip})
	} else {
		tools.Exit("not support windows")
	}
}

func DeleteNetWork(record *config.Record) {
	var ip []string

	for i := 0; i < len(record.Services); i++ {
		if record.Services[i].SelectPod == nil {
			continue
		}

		if record.Services[i].Switch != nil {
			ip = append(ip, record.Services[i].Switch.Pod.IP)
		}

		ip = append(ip, record.Services[i].SelectPod.IP)
	}

	if runtime.GOOS == "linux" {
		deleteLinux(ip)
	} else if runtime.GOOS == "darwin" {
		deleteDarwin(ip)
	} else {
		tools.Exit("not support windows")
	}
}

func createLinux(ip []string) {
	// ifconfig eth0:0 192.168.0.100 netmask 255.255.255.0 up
	// 255.255.255.0 -> just set one, but other can ping
	// 255.255.255.255 -> infinite, by other can not ping
	for i := 0; i < len(ip); i++ {
		var err = ExecCmd("ifconfig", fmt.Sprintf("eth0:%d", 100+i), ip[i], "netmask", "255.255.255.255", "up")
		if err != nil {
			console.Error("network: ip", ip[i], "create failed: ", err)
		} else {
			console.Info("network: ip", ip[i], "create success")
		}
	}
}

func createDarwin(ip []string) {
	// sudo ifconfig en0 alias 192.168.0.100 netmask 255.255.255.0 up
	for i := 0; i < len(ip); i++ {
		var err = ExecCmd("ifconfig", "en0", "alias", ip[i], "netmask", "255.255.255.255", "up")
		if err != nil {
			console.Error("network: ip", ip[i], "create failed: ", err)
		} else {
			console.Info("network: ip", ip[i], "create success")
		}
	}
}

func deleteLinux(ip []string) {
	// ifconfig eth0:0 del 192.168.0.100
	for i := 0; i < len(ip); i++ {
		var err = ExecCmd("ifconfig", fmt.Sprintf("eth0:%d", 100+i), "del", ip[i])
		if err != nil {
			console.Error("network: ip", ip[i], "delete failed: ", err)
		} else {
			console.Warning("network: ip", ip[i], "delete success")
		}
	}
}

func deleteDarwin(ip []string) {
	// sudo ifconfig en0 alias delete 192.168.0.100
	for i := 0; i < len(ip); i++ {
		var err = ExecCmd("ifconfig", "en0", "alias", "delete", ip[i])
		if err != nil {
			console.Error("network: ip", ip[i], "delete failed: ", err)
		} else {
			console.Warning("network: ip", ip[i], "delete success")
		}
	}
}

func ExecCmd(c string, args ...string) error {
	cmd := exec.Command(c, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
