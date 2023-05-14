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
	"github.com/lemonyxk/k8s-forward/services"
	"github.com/lemonyxk/k8s-forward/utils"
	"github.com/olekukonko/ts"
)

type Box struct {
	Pod *services.Pod
	Svc *services.Service
}

func Render() {
	// render

	// get terminal size
	size, err := ts.GetSize()
	if err != nil {
		console.Exit(err)
	}

	// console.Info(size.Col(), size.Row())

	if size.Col() < 180 {
		renderSmall()
	} else {
		renderBig()
	}
}

func renderSmall() {
	var table = console.NewTable()

	table.Header("SERVICE NAME", "POD NAME", "CLUSTER IP", "POD IP", "TYPE", "PORT")

	var boxes = make(map[string][]*Box)

	app.Services.Range(func(name string, service *services.Service) bool {
		service.Pods.Range(func(name string, pod *services.Pod) bool {
			boxes[service.Namespace] = append(boxes[service.Namespace], &Box{Svc: service, Pod: pod})
			return true
		})
		return true
	})

	var index = 0

	for namespace := range boxes {
		for i := 0; i < len(boxes[namespace]); i++ {
			var box = boxes[namespace][i]
			table.Row(
				box.Svc.Name,
				box.Pod.Name,
				box.Svc.ClusterIP,
				box.Pod.IP,
				box.Svc.Type,
				utils.ServicePortToString(box.Svc.Port),
			)
		}

		index++

		if index != len(boxes) {
			table.Row("-", "-", "-", "-", "-", "-")
		}
	}

	console.FgRed.Println(table.SortByName("NAMESPACE", 1).Render())
}

func renderBig() {
	var table = console.NewTable()

	table.Header("NAMESPACE", "SERVICE NAME", "POD NAME", "AGE", "RTS", "PHASE", "CLUSTER IP", "POD IP", "TYPE", "PORT")

	var boxes = make(map[string][]*Box)

	app.Services.Range(func(name string, service *services.Service) bool {
		service.Pods.Range(func(name string, pod *services.Pod) bool {
			boxes[service.Namespace] = append(boxes[service.Namespace], &Box{Svc: service, Pod: pod})
			return true
		})
		return true
	})

	var index = 0

	for namespace := range boxes {
		for i := 0; i < len(boxes[namespace]); i++ {
			var box = boxes[namespace][i]
			table.Row(
				box.Svc.Namespace,
				box.Svc.Name,
				box.Pod.Name,
				GetAge(box.Pod.Age),
				box.Pod.Restarts,
				box.Pod.Phase,
				box.Svc.ClusterIP,
				box.Pod.IP,
				box.Svc.Type,
				utils.ServicePortToString(box.Svc.Port),
			)
		}

		index++

		if index != len(boxes) {
			table.Row("-", "-", "-", "-", "-", "-", "-", "-", "-", "-")
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
