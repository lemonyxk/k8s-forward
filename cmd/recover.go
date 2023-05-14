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
	"fmt"
	"os"
	"strings"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/k8s"
	"github.com/lemonyxk/k8s-forward/services"
	"github.com/lemonyxk/k8s-forward/utils"
	v1 "k8s.io/api/core/v1"
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

	resource = strings.ToLower(resource)

	var scale, err = k8s.GetScale(resource, namespace, name)
	if err != nil {
		return err.Error()
	}

	// match service
	var svc *services.Service
	app.Services.Range(func(name string, service *services.Service) bool {
		if utils.Match(utils.MakeLabels(scale.Status.Selector), service.Selector) {
			svc = service
			return false
		}
		return true
	})

	if svc == nil {
		return fmt.Sprintf("%s %s %s not match service", resource, namespace, name)
	}

	if svc.Switch == nil {
		return fmt.Sprintf("%s %s %s has not switch", resource, namespace, name)
	}

	if svc.Switch.Pod == nil {
		return fmt.Sprintf("%s %s %s has not switch pod", resource, namespace, name)
	}

	if svc.Switch.Deployment == nil {
		return fmt.Sprintf("%s %s %s has not deployment", resource, namespace, name)
	}

	if svc.Switch.Scale == nil {
		return fmt.Sprintf("%s %s %s has not scale", resource, namespace, name)
	}

	var podNum = svc.Switch.Scale.Spec.Replicas

	console.Info("match service:", svc.Name, "replicas:", scale.Spec.Replicas)

	var pods []*v1.Pod
	var ch = make(chan struct{})
	go func() {
		pods = <-app.Watch.Watch(&app.Filter{
			Namespace: svc.Namespace,
			Selector:  svc.Selector,
			Name:      scale.Name,
			Number:    podNum,
		})
		ch <- struct{}{}
	}()

	err = k8s.UnSwitchScale(svc)
	if err != nil {
		return err.Error()
	}

	svc.Switch.Scale = nil

	app.SaveAllServices(app.Services)

	<-ch

	svc.Switch.StopForward <- struct{}{}
	svc.Switch.StopSSH <- struct{}{}

	err = k8s.UnSwitchDeployment(svc)
	if err != nil {
		return err.Error()
	}

	for i := 0; i < len(pods); i++ {
		var pod = &services.Pod{
			Namespace:   pods[i].Namespace,
			Name:        pods[i].Name,
			IP:          pods[i].Status.PodIP,
			Labels:      pods[i].Labels,
			HostNetwork: pods[i].Spec.HostNetwork,
			Age:         pods[i].CreationTimestamp.Time,
			Restarts:    pods[i].Status.ContainerStatuses[0].RestartCount,
			Phase:       pods[i].Status.Phase,
			Containers:  pods[i].Spec.Containers,
			Forwarded:   false,
			StopForward: nil,
		}

		svc.Pods.Set(pod.Name, pod)
	}

	svc.Switch.Deployment = nil

	svc.Switch.Pod = nil

	svc.Switch = nil

	app.SaveAllServices(app.Services)

	console.Warning("recover", resource, namespace, name, "success")

	return fmt.Sprintf("%s %s %s recover success", resource, namespace, name)
}
