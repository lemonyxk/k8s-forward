/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-09 12:11
**/

package utils

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	v12 "k8s.io/api/core/v1"
)

func ReplaceString(s string, oList []string, nList []string) string {
	for i := 0; i < len(oList); i++ {
		s = strings.ReplaceAll(s, oList[i], nList[i])
	}
	return s
}

func GetArgs(flag ...string) string {
	var args = os.Args[1:]
	for i := 0; i < len(args); i++ {
		for j := 0; j < len(flag); j++ {
			if args[i] == flag[j] {
				if i+1 < len(args) {
					return args[i+1]
				}
			}
		}
	}
	return ""
}

func GetMultiArgs(flag ...string) []string {
	var args = os.Args[1:]
	var res []string
	for i := 0; i < len(args); i++ {
		for j := 0; j < len(flag); j++ {
			if args[i] == flag[j] {
				if i+1 < len(args) {
					res = append(res, args[i+1])
					for {
						i++
						if i+1 >= len(args) {
							break
						}
						if strings.HasPrefix(args[i+1], "-") {
							break
						}
						res = append(res, args[i+1])
					}
				}
			}
		}
	}
	return res
}

func GetFlagAndArgs(flag ...string) (string, string) {
	var args = os.Args[1:]
	for i := 0; i < len(args); i++ {
		for j := 0; j < len(flag); j++ {
			if args[i] == flag[j] {
				if i+1 < len(args) {
					return flag[j], args[i+1]
				}
			}
		}
	}
	return "", ""
}

func HasArgs(flag ...string) bool {
	var args = os.Args[1:]
	for i := 0; i < len(args); i++ {
		for j := 0; j < len(flag); j++ {
			if args[i] == flag[j] {
				return true
			}
		}
	}
	return false
}

func GetServerPorts(serverPorts []v12.ServicePort) []string {
	var ports []string
	for i := 0; i < len(serverPorts); i++ {
		ports = append(ports, fmt.Sprintf("%d:%s", serverPorts[i].Port, serverPorts[i].TargetPort.String()))
	}
	return ports
}

func ServicePortToString(serverPorts []v12.ServicePort) string {
	var ports []string
	for i := 0; i < len(serverPorts); i++ {
		ports = append(ports, fmt.Sprintf("%d:%s/%s", serverPorts[i].Port, serverPorts[i].TargetPort.String(), serverPorts[i].Protocol))
	}
	return strings.Join(ports, ",")
}

func Match(labels map[string]string, selector map[string]string) bool {
	for k1, v1 := range labels {
		if selector[k1] == v1 {
			return true
		}
	}
	return false
}

func MakeLabels(str string) map[string]string {
	var res = make(map[string]string)
	var arr = strings.Split(str, ",")
	for i := 0; i < len(arr); i++ {
		var v = strings.Split(arr[i], "=")
		if len(v) != 2 {
			continue
		}
		res[v[0]] = v[1]
	}

	return res
}

func MakeLabelsString(labels map[string]string) string {
	var res = ""
	for k, v := range labels {
		res += k + "=" + v + ","
	}
	return strings.TrimRight(res, ",")
}

func RandomString(n int) string {
	var letter = []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	var res []byte
	for i := 0; i < n; i++ {
		res = append(res, letter[rand.Intn(len(letter))])
	}
	return string(res)
}
