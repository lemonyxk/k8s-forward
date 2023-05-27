/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-10 00:40
**/

package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/k8s"
	"github.com/lemonyxk/k8s-forward/services"
	"github.com/lemonyxk/k8s-forward/ssh"
	"github.com/lemonyxk/k8s-forward/utils"
	utils2 "github.com/lemoyxk/utils"
	v1 "k8s.io/api/core/v1"
)

func Switch() string {

	var resource = os.Args[2]
	var name = os.Args[3]

	var namespace = utils.GetArgs("--namespace", "-n")
	if namespace == "" {
		namespace = "default"
	}

	var image = utils.GetArgs("--image", "-i")
	if image == "" {
		image = `a1354243/root-ssh-server:latest`
	}

	var port = 0
	var err error
	var portStr = utils.GetArgs("-p", "--port")
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return "switch port is not number"
		}
	}

	return doSwitch(resource, namespace, name, port, image)
}

func doSwitch(resource string, namespace string, name string, port int, image string) string {

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

	if svc.Switch != nil {
		return fmt.Sprintf("%s %s %s has switch", resource, namespace, name)
	}

	svc.Switch = &services.Switch{}

	console.Info("match service:", svc.Name, "replicas:", scale.Spec.Replicas)

	_, err = k8s.Scale(scale, 0)
	if err != nil {
		return err.Error()
	}

	svc.Switch.Scale = scale

	app.SaveAllServices(app.Services)

	deployment, err := k8s.GenerateDeployment()
	if err != nil {
		return err.Error()
	}

	deployment.ObjectMeta.Name = svc.Name + "-" + utils.RandomString(4)
	deployment.ObjectMeta.Namespace = svc.Namespace
	deployment.ObjectMeta.Labels = svc.Selector
	deployment.Spec.Selector.MatchLabels = svc.Selector
	deployment.Spec.Template.ObjectMeta.Labels = svc.Selector
	deployment.Spec.Template.Spec.Containers[0].Image = image
	deployment.Spec.Template.Spec.Containers[0].Name = svc.Name
	deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = 22
	deployment.Spec.Template.Spec.Containers[0].Ports[0].Name = "ssh"

	var ch = make(chan struct{})
	var pods []*v1.Pod
	go func() {
		pods = <-app.Watch.Watch(&app.Filter{
			Namespace: svc.Namespace,
			Selector:  svc.Selector,
			Name:      deployment.Name,
			Number:    1,
		})
		ch <- struct{}{}
	}()

	_, err = k8s.Deployment(deployment)
	if err != nil {
		return err.Error()
	}

	svc.Switch.Deployment = deployment

	app.SaveAllServices(app.Services)

	<-ch

	pod := pods[0]

	svc.Switch.Pod = &services.Pod{
		Namespace:   svc.Namespace,
		Name:        pod.ObjectMeta.Name,
		IP:          pod.Status.PodIP,
		Labels:      pod.Labels,
		HostNetwork: pod.Spec.HostNetwork,
		Age:         pod.CreationTimestamp.Time,
		Restarts:    pod.Status.ContainerStatuses[0].RestartCount,
		Phase:       pod.Status.Phase,
		Containers:  pod.Spec.Containers,
		Forwarded:   false,
		StopForward: nil,
	}

	app.SaveAllServices(app.Services)

	readyPod, stopPod, err := k8s.ForwardPod(svc.Switch.Pod, []string{"0.0.0.0"}, []string{"2222:2222"})
	if err != nil {
		return err.Error()
	}

	<-readyPod

	svc.Switch.StopForward = stopPod

	// ssh
	var remoteAddr = fmt.Sprintf("%s:%d", pod.Status.PodIP, svc.Port[0].Port)
	var localAddr = fmt.Sprintf("%s:%d", "127.0.0.1", utils2.Ternary.Int(port == 0, int(svc.Port[0].Port), port))
	stopSSH, _, err := ssh.RemoteForward(ssh.ForwardConfig{
		UserName:          "root",
		Password:          "root",
		PrivateKey:        "",
		ServerAddress:     "127.0.0.1:2222",
		RemoteAddress:     remoteAddr,
		LocalAddress:      localAddr,
		Timeout:           time.Second * 3,
		HeartbeatInterval: time.Second,
	})
	if err != nil {
		return err.Error()
	}

	svc.Switch.StopSSH = stopSSH

	console.Warning("switch", resource, namespace, name, "success")

	return fmt.Sprintf("switch %s %s %s success", resource, namespace, name)
}
