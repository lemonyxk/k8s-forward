/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-08 17:22
**/

package k8s

import (
	"fmt"
	"strings"
	"time"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemoyxk/utils"
)

func Render() {
	// render
	var table = console.NewTable()

	table.Header("NAMESPACE", "TYPE", "SERVICE NAME", "POD NAME", "AGE", "RESTARTS", "CLUSTER IP", "POD IP", "POD PORT")

	var servicesMap = make(map[string][]*config.Service)
	var servicesList [][]*config.Service

	for i := 0; i < len(app.Record.Services); i++ {
		if len(app.Record.Services[i].Pod) == 0 {
			continue
		}

		servicesMap[app.Record.Services[i].Namespace] =
			append(servicesMap[app.Record.Services[i].Namespace], app.Record.Services[i])
	}

	for namespace := range servicesMap {
		servicesList = append(servicesList, servicesMap[namespace])
	}

	for i := 0; i < len(servicesList); i++ {
		for j := 0; j < len(servicesList[i]); j++ {
			for k := 0; k < len(servicesList[i][j].Pod); k++ {
				var svc = servicesList[i][j]
				var pod = svc.Pod[k]

				table.Row(
					svc.Namespace,
					svc.Type,
					svc.Name,
					pod.Name,
					GetAge(pod.Age),
					pod.Restarts,
					svc.ClusterIP,
					pod.IP,
					strings.Join(utils.Extract.Src(svc.Port).Field("Port").String(), ","),
				)

			}
		}
		if i != len(servicesList)-1 {
			table.Row("-", "-", "-", "-", "-", "-", "-", "-")
		}
	}

	console.FgRed.Println(table.SortByName("NAMESPACE", 1).Render())
}

func GetAge(start time.Time) string {
	// < 60s
	var sub = time.Now().Sub(start)
	if sub < time.Second*60 {
		return fmt.Sprintf("%ds", sub/time.Second)
	}

	// < 60m
	if sub < time.Minute*60 {
		return fmt.Sprintf("%dm", sub/time.Minute)
	}

	// < 24h
	if sub < time.Hour*24 {
		return fmt.Sprintf("%dh", sub/time.Hour)
	}

	return fmt.Sprintf("%dd", sub/(time.Hour*24))
}
