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
	"net/url"
	"os"
	"os/exec"
	"runtime"

	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemoyxk/console"
	"github.com/lemoyxk/utils"
)

func CreateNetWorkByIp(pod *config.Pod) {
	if IsLocal() && pod.HostNetwork {
		return
	}
	if runtime.GOOS == "linux" {
		createLinux([]string{pod.IP})
	} else if runtime.GOOS == "darwin" {
		createDarwin([]string{pod.IP})
	} else {
		console.Exit("not support windows")
	}
}

var isLocal *bool

func IsLocal() bool {
	if isLocal != nil {
		return *isLocal
	}
	var appHost = app.RestConfig.Host
	u, err := url.Parse(appHost)
	if err != nil {
		panic(err)
	}
	var res = utils.Addr.IsLocalIP(u.Hostname())
	isLocal = &res
	return *isLocal
}

func CreateNetWork(record *config.Record) {

	var ip []string

	for i := 0; i < len(record.Services); i++ {

		if record.Services[i].Pod == nil {
			continue
		}

		if record.Services[i].Switch != nil {
			if !IsLocal() || !record.Services[i].Switch.Pod.HostNetwork {
				ip = append(ip, record.Services[i].Switch.Pod.IP)
			}
		}

		if !IsLocal() || !record.Services[i].Pod.HostNetwork {
			ip = append(ip, record.Services[i].Pod.IP)
		}
	}

	if runtime.GOOS == "linux" {
		createLinux(ip)
	} else if runtime.GOOS == "darwin" {
		createDarwin(ip)
	} else {
		console.Exit("not support windows")
	}
}

func DeleteNetWorkByIp(pod *config.Pod) {
	if IsLocal() && pod.HostNetwork {
		return
	}
	if runtime.GOOS == "linux" {
		deleteLinux([]string{pod.IP})
	} else if runtime.GOOS == "darwin" {
		deleteDarwin([]string{pod.IP})
	} else {
		console.Exit("not support windows")
	}
}

func DeleteNetWork(record *config.Record) {
	var ip []string

	for i := 0; i < len(record.Services); i++ {
		if record.Services[i].Pod == nil {
			continue
		}

		if record.Services[i].Switch != nil {
			if !IsLocal() || !record.Services[i].Switch.Pod.HostNetwork {
				ip = append(ip, record.Services[i].Switch.Pod.IP)
			}
		}

		if !IsLocal() || !record.Services[i].Pod.HostNetwork {
			ip = append(ip, record.Services[i].Pod.IP)
		}
	}

	if runtime.GOOS == "linux" {
		deleteLinux(ip)
	} else if runtime.GOOS == "darwin" {
		deleteDarwin(ip)
	} else {
		console.Exit("not support windows")
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

func createWindows(ip []string) {
	// netsh interface ip add address "WI-FI" 192.168.0.100 255.255.255.255
	// you need get interface first
	// netsh interface show interface
	var interfaceName = "WI-FI"
	for i := 0; i < len(ip); i++ {
		var err = ExecCmd("netsh", "interface", "ip", "add", "address", interfaceName, ip[i], "255.255.255.255")
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

func deleteWindows(ip []string) {
	// netsh interface ip delete address "WI-FI" 192.168.0.100
	// you need get interface first
	// netsh interface show interface
	var interfaceName = "WI-FI"
	for i := 0; i < len(ip); i++ {
		var err = ExecCmd("netsh", "interface", "ip", "delete", "address", interfaceName, ip[i])
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
