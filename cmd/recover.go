/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-12 22:55
**/

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemonyxk/k8s-forward/k8s"
	"github.com/lemonyxk/k8s-forward/net"
	"github.com/lemonyxk/k8s-forward/utils"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func Recover() string {
	var resource = os.Args[2]
	var name = os.Args[3]

	var namespace = utils.GetArgs("--namespace", "-n")
	if namespace == "" {
		namespace = "default"
	}

	return doRecover(resource, namespace, name)
}

func doRecover(resource string, namespace string, name string) string {
	var client = app.Client

	resource = strings.ToLower(resource)

	var scale, err = GetScale(resource, namespace, name)
	if err != nil {
		return err.Error()
	}

	// match service
	var service *config.Service
	for i := 0; i < len(app.Record.Services); i++ {
		if k8s.Match(k8s.MakeLabels(scale.Status.Selector), app.Record.Services[i].Selector) {
			service = app.Record.Services[i]
			break
		}
	}

	if service == nil {
		return fmt.Sprintf("%s %s %s not match service", resource, namespace, name)
	}

	if service.Switch == nil {
		return fmt.Sprintf("%s %s %s has not switch", resource, namespace, name)
	}

	if service.Switch.Pod == nil {
		return fmt.Sprintf("%s %s %s has not switch pod", resource, namespace, name)
	}

	if service.Switch.Deployment == nil {
		return fmt.Sprintf("%s %s %s has not deployment", resource, namespace, name)
	}

	if service.Switch.Scale == nil {
		return fmt.Sprintf("%s %s %s has not scale", resource, namespace, name)
	}

	var podNum = service.Switch.Scale.Spec.Replicas

	console.Info("match service:", service.Name, "replicas:", scale.Spec.Replicas, podNum)

	err = UnScale(service)
	if err != nil {
		return err.Error()
	}

	service.Switch.Scale = nil

	k8s.SaveRecordToFile(app.Record)

	watch, err := client.CoreV1().Pods(service.Namespace).Watch(context.Background(), metav1.ListOptions{
		Watch:         true,
		LabelSelector: labels.Set(service.Selector).AsSelector().String(),
	})
	if err != nil {
		return err.Error()
	}

	var ch = make(chan struct{})
	var pod []*v12.Pod

	go func() {
		for {
			select {
			case event, ok := <-watch.ResultChan():
				if !ok {
					ch <- struct{}{}
					return
				}
				p, ok := event.Object.(*v12.Pod)
				if !ok {
					continue
				}

				if strings.HasPrefix(p.Name, scale.Name) {
					if p.Status.Phase == v12.PodRunning && p.Namespace == service.Namespace {
						pod = append(pod, p)
						console.Info("new pod:", p.Name, "ip:", p.Status.PodIP)
						if len(pod) == int(podNum) {
							watch.Stop()
						}
					}
				}
			}
		}
	}()

	<-ch

	service.Switch.StopForward <- struct{}{}
	service.Switch.StopSSH <- struct{}{}

	err = UnDeployment(service)
	if err != nil {
		return err.Error()
	}

	service.Switch.Status = config.Stop

	for i := 0; i < len(pod); i++ {
		service.Pod = append(service.Pod, &config.Pod{
			Namespace:   pod[i].Namespace,
			Name:        pod[i].Name,
			IP:          pod[i].Status.PodIP,
			Labels:      pod[i].Labels,
			HostNetwork: pod[i].Spec.HostNetwork,
			Age:         pod[i].CreationTimestamp.Time,
			Restarts:    pod[i].Status.ContainerStatuses[0].RestartCount,
			Phase:       pod[i].Status.Phase,
			Containers:  pod[i].Spec.Containers,
		})
	}

	for i := 0; i < len(service.Pod); i++ {
		net.CreateNetWorkByIp(service.Pod[i])
	}

	// delete switch pod ip
	app.Record.History = append(app.Record.History, service.Switch.Pod)

	service.Switch.Deployment = nil

	service.Switch.Pod = nil

	service.Switch = nil

	k8s.SaveRecordToFile(app.Record)

	console.Warning("recover", resource, namespace, name, "success")

	return fmt.Sprintf("%s %s %s recover success", resource, namespace, name)
}
