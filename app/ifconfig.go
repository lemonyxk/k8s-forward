/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-08 18:18
**/

package app

import (
	"net/url"
	"os"
	"os/exec"
	"runtime"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/services"
	"github.com/lemoyxk/utils"
	v1 "k8s.io/api/core/v1"
)

func CreateNetWorkByPod(pod *v1.Pod) {
	if IsLocal() && pod.Spec.HostNetwork {
		return
	}

	switch runtime.GOOS {
	case "linux":
		console.Exit("not support linux")
	case "darwin":
		createDarwin(pod.Status.PodIP)
	default:
		console.Exit("not support windows")
	}
}

var isLocal *bool

func IsLocal() bool {
	if isLocal != nil {
		return *isLocal
	}
	var appHost = RestConfig.Host
	u, err := url.Parse(appHost)
	if err != nil {
		panic(err)
	}
	var res = utils.Addr.IsLocalIP(u.Hostname())
	isLocal = &res
	return *isLocal
}

func CreateNetWork(svs *services.Services) {
	var ips []string

	var history = svs.History()

	for i := 0; i < len(history); i++ {
		if !IsLocal() || !history[i].HostNetwork {
			ips = append(ips, history[i].IP)
		}
	}

	switch runtime.GOOS {
	case "linux":
		console.Exit("not support linux")
	case "darwin":
		createDarwin(ips...)
	default:
		console.Exit("not support windows")
	}
}

func DeleteNetWork(svs *services.Services) {
	var ips []string

	var history = svs.History()

	for i := 0; i < len(history); i++ {
		if !IsLocal() || !history[i].HostNetwork {
			ips = append(ips, history[i].IP)
		}
	}

	switch runtime.GOOS {
	case "linux":
		console.Exit("not support linux")
	case "darwin":
		deleteDarwin(ips...)
	default:
		console.Exit("not support windows")
	}
}

// func createLinux(ip []string) {
// 	// ifconfig eth0:0 192.168.0.100 netmask 255.255.255.0 up
// 	// 255.255.255.0 -> just set one, but other can ping
// 	// 255.255.255.255 -> infinite, by other can not ping
// 	for i := 0; i < len(ip); i++ {
// 		var err = ExecCmd("ifconfig", fmt.Sprintf("eth0:%d", 100+i), ip[i], "netmask", "255.255.255.255", "up")
// 		if err != nil {
// 			console.Error("network: ip", ip[i], "create failed: ", err)
// 		} else {
// 			// console.Info("network: ip", ip[i], "create success")
// 		}
// 	}
// }

func createDarwin(ips ...string) {
	var hasCreate = make(map[string]bool)
	// sudo ifconfig en0 alias 192.168.0.100 netmask 255.255.255.0 up
	for i := 0; i < len(ips); i++ {
		if hasCreate[ips[i]] {
			continue
		}
		if ips[i] == "" {
			continue
		}
		var err = ExecCmd("ifconfig", "en0", "alias", ips[i], "netmask", "255.255.255.255", "up")
		if err != nil {
			console.Error("network: ip", ips[i], "create failed: ", err)
		} else {
			hasCreate[ips[i]] = true
			// console.Info("network: ip", ip[i], "create success")
		}
	}
}

// func createWindows(ip []string) {
// 	// netsh interface ip add address "WI-FI" 192.168.0.100 255.255.255.255
// 	// you need get interface first
// 	// netsh interface show interface
// 	var interfaceName = "WI-FI"
// 	for i := 0; i < len(ip); i++ {
// 		var err = ExecCmd("netsh", "interface", "ip", "add", "address", interfaceName, ip[i], "255.255.255.255")
// 		if err != nil {
// 			console.Error("network: ip", ip[i], "create failed: ", err)
// 		} else {
// 			// console.Info("network: ip", ip[i], "create success")
// 		}
// 	}
// }
//
// func deleteLinux(ip []string) {
// 	// ifconfig eth0:0 del 192.168.0.100
// 	for i := 0; i < len(ip); i++ {
// 		var err = ExecCmd("ifconfig", fmt.Sprintf("eth0:%d", 100+i), "del", ip[i])
// 		if err != nil {
// 			console.Error("network: ip", ip[i], "delete failed: ", err)
// 		} else {
// 			// console.Warning("network: ip", ip[i], "delete success")
// 		}
// 	}
// }

func deleteDarwin(ips ...string) {
	var hasDelete = make(map[string]bool)
	// sudo ifconfig en0 alias delete 192.168.0.100
	for i := 0; i < len(ips); i++ {
		if hasDelete[ips[i]] {
			continue
		}
		if ips[i] == "" {
			continue
		}
		var err = ExecCmd("ifconfig", "en0", "alias", "delete", ips[i])
		if err != nil {
			console.Error("network: ip", ips[i], "delete failed: ", err)
		} else {
			hasDelete[ips[i]] = true
			// console.Warning("network: ip", ip[i], "delete success")
		}
	}
}

// func deleteWindows(ip []string) {
// 	// netsh interface ip delete address "WI-FI" 192.168.0.100
// 	// you need get interface first
// 	// netsh interface show interface
// 	var interfaceName = "WI-FI"
// 	for i := 0; i < len(ip); i++ {
// 		var err = ExecCmd("netsh", "interface", "ip", "delete", "address", interfaceName, ip[i])
// 		if err != nil {
// 			console.Error("network: ip", ip[i], "delete failed: ", err)
// 		} else {
// 			// console.Warning("network: ip", ip[i], "delete success")
// 		}
// 	}
// }

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
