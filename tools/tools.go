/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-09 12:11
**/

package tools

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lemonyxk/k8s-forward/app"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
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

func GenerateDeployment() (*v1.Deployment, error) {
	var f, err = app.Temp.Open("temp/deployment.yaml")
	if err != nil {
		return nil, err
	}

	bts, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var deployment v1.Deployment

	err = yaml.Unmarshal(bts, &deployment)
	if err != nil {
		return nil, err
	}

	return &deployment, nil
}

func GetServerPorts(serverPorts []v12.ServicePort) []string {
	var ports []string
	for i := 0; i < len(serverPorts); i++ {
		ports = append(ports, fmt.Sprintf("%d:%d", serverPorts[i].Port, serverPorts[i].TargetPort))
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
