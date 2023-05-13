/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-08 17:22
**/

package k8s

import (
	"fmt"
	"time"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemonyxk/k8s-forward/utils"
	"github.com/olekukonko/ts"
)

func Render() {
	// render

	// get terminal size
	size, err := ts.GetSize()
	if err != nil {
		console.Exit(err)
	}

	console.Info(fmt.Sprintf("terminal size: %d x %d", size.Col(), size.Row()))

	if size.Col() < 180 {
		renderSmall()
	} else {
		renderBig()
	}
}

func renderSmall() {
	var table = console.NewTable()

	table.Header("SERVICE NAME", "POD NAME", "CLUSTER IP", "POD IP", "TYPE", "PORT")

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
					svc.Name,
					pod.Name,
					svc.ClusterIP,
					pod.IP,
					svc.Type,
					utils.ServicePortToString(svc.Port),
				)
			}
		}
		if i != len(servicesList)-1 {
			table.Row("-", "-", "-", "-", "-", "-")
		}
	}

	console.FgRed.Println(table.SortByName("NAMESPACE", 1).Render())
}

func renderBig() {
	var table = console.NewTable()

	table.Header("NAMESPACE", "SERVICE NAME", "POD NAME", "AGE", "RTS", "PHASE", "CLUSTER IP", "POD IP", "TYPE", "PORT")

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
					svc.Name,
					pod.Name,
					GetAge(pod.Age),
					pod.Restarts,
					pod.Phase,
					svc.ClusterIP,
					pod.IP,
					svc.Type,
					utils.ServicePortToString(svc.Port),
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
