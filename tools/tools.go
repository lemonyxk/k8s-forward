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

	"github.com/lemoyxk/console"
	"github.com/lemoyxk/k8s-forward/app"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var bin = ""

func GetBin() string {
	if bin == "" {
		// Figure out our executable
		var err error
		bin, err = os.Executable()
		if err != nil {
			Exit(err)
		}
	}
	return bin
}

func ReplaceString(s string, oList []string, nList []string) string {
	for i := 0; i < len(oList); i++ {
		s = strings.ReplaceAll(s, oList[i], nList[i])
	}
	return s
}

func Exit(args ...interface{}) {
	console.Error(args...)
	os.Exit(0)
}

func GetArgs(flag []string, args []string) string {
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

func HasArgs(flag string, args []string) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == flag {
			return true
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
		ports = append(ports, fmt.Sprintf("%d:%d", serverPorts[i].Port, serverPorts[i].Port))
	}
	return ports
}
