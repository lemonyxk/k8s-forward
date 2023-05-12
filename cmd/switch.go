/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-10 00:40
**/

package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemonyxk/k8s-forward/k8s"
	"github.com/lemonyxk/k8s-forward/net"
	"github.com/lemonyxk/k8s-forward/ssh"
	"github.com/lemonyxk/k8s-forward/tools"
	"github.com/lemoyxk/utils"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func Switch() string {

	var resource = os.Args[2]
	var name = os.Args[3]

	var namespace = tools.GetArgs("--namespace", "-n")
	if namespace == "" {
		namespace = "default"
	}

	var image = tools.GetArgs("--image", "-i")
	if image == "" {
		image = `a1354243/root-ssh-server:latest`
	}

	var port = 0
	var err error
	var portStr = tools.GetArgs("-p", "--port")
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return "switch port is not number"
		}
	}

	return doSwitch(resource, namespace, name, port, image)
}

func doSwitch(resource string, namespace string, name string, port int, image string) string {

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

	if service.Switch != nil {
		return fmt.Sprintf("%s %s %s has switch", resource, namespace, name)
	}

	service.Switch = &config.Switch{}

	console.Info("match service:", service.Name, "replicas:", scale.Spec.Replicas)

	if int(service.ForwardNumber) == len(service.Pod) {
		for i := 0; i < len(service.StopForward); i++ {
			service.StopForward[i] <- struct{}{}
		}
	}

	// delete old pod ip
	for i := 0; i < len(service.Pod); i++ {
		app.Record.History = append(app.Record.History, service.Pod[i])
	}

	service.Pod = nil

	us, err := UpdateScale(scale, 0)
	if err != nil {
		return err.Error()
	}

	service.Switch.Scale = scale

	k8s.SaveRecordToFile(app.Record)

	console.Info("update service:", service.Name, "replicas:", us.Spec.Replicas)

	deployment, err := tools.GenerateDeployment()
	if err != nil {
		return err.Error()
	}

	deployment.ObjectMeta.Name = service.Name + "-" + utils.Rand.UUID()
	deployment.ObjectMeta.Namespace = service.Namespace
	deployment.ObjectMeta.Labels = service.Selector
	deployment.Spec.Template.ObjectMeta.Labels = service.Selector
	deployment.Spec.Selector.MatchLabels = service.Selector
	deployment.Spec.Template.Spec.Containers[0].Image = image

	_, err = client.AppsV1().Deployments(service.Namespace).Create(context.Background(), deployment, v1.CreateOptions{})
	if err != nil {
		return err.Error()
	}

	service.Switch.Deployment = deployment

	k8s.SaveRecordToFile(app.Record)

	watch, err := client.CoreV1().Pods(service.Namespace).Watch(context.Background(), v1.ListOptions{
		Watch:         true,
		LabelSelector: labels.Set(service.Selector).AsSelector().String(),
	})
	if err != nil {
		return err.Error()
	}

	var readyPod = make(chan struct{})
	var pod *v12.Pod

	go func() {
		for {
			select {
			case event, ok := <-watch.ResultChan():
				if !ok {
					readyPod <- struct{}{}
					return
				}
				p, ok := event.Object.(*v12.Pod)
				if !ok {
					continue
				}

				if strings.HasPrefix(p.Name, deployment.ObjectMeta.Name) {
					if p.Status.Phase == v12.PodRunning && p.Namespace == service.Namespace {
						pod = p
						watch.Stop()
					}
				}
			}
		}
	}()

	<-readyPod

	console.Info("create deployment:", deployment.ObjectMeta.Name)

	service.Switch.Pod = &config.Pod{
		Namespace:   service.Namespace,
		Name:        pod.ObjectMeta.Name,
		IP:          pod.Status.PodIP,
		Labels:      pod.Labels,
		HostNetwork: pod.Spec.HostNetwork,
		Age:         pod.CreationTimestamp.Time,
		Restarts:    pod.Status.ContainerStatuses[0].RestartCount,
		Phase:       pod.Status.Phase,
	}

	net.CreateNetWorkByIp(service.Switch.Pod)

	k8s.SaveRecordToFile(app.Record)

	readyPod, stopPod, err := k8s.ForwardPod(namespace, pod.ObjectMeta.Name, []string{"0.0.0.0"}, []string{"2222:2222"})
	if err != nil {
		return err.Error()
	}

	<-readyPod

	service.Switch.StopForward = stopPod

	// ssh
	var remoteAddr = fmt.Sprintf("%s:%d", pod.Status.PodIP, service.Port[0].Port)
	var localAddr = fmt.Sprintf("%s:%d", "127.0.0.1", utils.Ternary.Int(port == 0, int(service.Port[0].Port), port))
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

	service.Switch.StopSSH = stopSSH

	console.Warning("switch", resource, namespace, name, "success")

	return fmt.Sprintf("switch %s %s %s success", resource, namespace, name)
}
