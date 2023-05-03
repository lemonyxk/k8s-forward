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
	v1 "k8s.io/api/autoscaling/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func Switch(args []string) string {

	var resource = tools.GetArgs([]string{"switch", "-w", "--switch"}, args)
	if resource == "" {
		return "switch resource is empty"
	}

	var name = tools.GetArgs([]string{resource}, args)
	if name == "" {
		return "switch resource name is empty"
	}

	var namespace = tools.GetArgs([]string{"--namespace", "-n"}, args)
	if namespace == "" {
		namespace = "default"
	}

	var port = 0
	var err error
	var portStr = tools.GetArgs([]string{"-p", "--port"}, args)
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return "switch port is not number"
		}
	}

	return doSwitch(resource, namespace, name, port)
}

func doSwitch(resource string, namespace string, name string, port int) string {

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

	if service.Status == config.Start {
		service.StopForward <- struct{}{}
	}

	// delete old pod ip
	net.DeleteNetWorkByIp(service.Pod)
	service.Pod = nil
	k8s.SaveRecordToFile(app.Record)

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

	deployment.ObjectMeta.Name = "ssh-server-" + utils.Rand.UUID()
	deployment.ObjectMeta.Namespace = service.Namespace
	deployment.ObjectMeta.Labels = service.Selector
	deployment.Spec.Template.ObjectMeta.Labels = service.Selector
	deployment.Spec.Selector.MatchLabels = service.Selector

	_, err = client.AppsV1().Deployments(service.Namespace).Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		return err.Error()
	}

	service.Switch.Deployment = deployment
	k8s.SaveRecordToFile(app.Record)

	watch, err := client.CoreV1().Pods(service.Namespace).Watch(context.Background(), metav1.ListOptions{
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
	}
	k8s.SaveRecordToFile(app.Record)

	net.CreateNetWorkByIp(service.Switch.Pod)

	readyPod, stopPod, err := k8s.ForwardPod(namespace, pod.ObjectMeta.Name, []string{"0.0.0.0"}, []string{"2222:2222"})
	if err != nil {
		return err.Error()
	}

	<-readyPod

	service.Switch.StopForward = stopPod

	// ssh
	var remoteAddr = fmt.Sprintf("%s:%d", pod.Status.PodIP, service.Port[0].Port)
	var localAddr = fmt.Sprintf("%s:%d", "127.0.0.1", utils.Ternary.Int(port == 0, int(service.Port[0].Port), port))
	stopSSH, _, err := ssh.RemoteForward(ssh.Config{
		UserName:      "root",
		Password:      "root",
		ServerAddress: "127.0.0.1:2222",
		RemoteAddress: remoteAddr,
		LocalAddress:  localAddr,
		Timeout:       time.Second * 3,
		// Reconnect:         0,
		HeartbeatInterval: time.Second,
	})
	if err != nil {
		return err.Error()
	}

	service.Switch.StopSSH = stopSSH

	return fmt.Sprintf("switch %s %s %s success", resource, namespace, name)
}

func GetScale(resource string, namespace string, name string) (*v1.Scale, error) {

	var client = app.Client

	resource = strings.ToLower(resource)

	var scale *v1.Scale

	switch resource {
	case "deployment":
		s, err := client.AppsV1().
			Deployments(namespace).
			GetScale(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		scale = s

	case "statefulset":
		s, err := client.AppsV1().StatefulSets(namespace).
			GetScale(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		scale = s

	case "daemonset":
		return nil, fmt.Errorf("daemonset not support")
	case "replicaset":
		s, err := client.AppsV1().ReplicaSets(namespace).
			GetScale(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		scale = s

	default:
		return nil, fmt.Errorf("%s not support", resource)
	}

	scale.Kind = resource

	return scale, nil
}

func UpdateScale(scale *v1.Scale, replicas int32) (*v1.Scale, error) {

	var client = app.Client

	sc := *scale
	sc.Spec.Replicas = replicas

	switch scale.Kind {
	case "deployment":
		return client.AppsV1().Deployments(scale.Namespace).UpdateScale(context.TODO(), scale.Name, &sc, metav1.UpdateOptions{})
	case "statefulset":
		return client.AppsV1().StatefulSets(scale.Namespace).UpdateScale(context.TODO(), scale.Name, &sc, metav1.UpdateOptions{})
	case "daemonset":
		return nil, fmt.Errorf("daemonset not support")
	case "replicaset":
		return client.AppsV1().ReplicaSets(scale.Namespace).UpdateScale(context.TODO(), scale.Name, &sc, metav1.UpdateOptions{})
	default:
		return nil, fmt.Errorf("%s not support", scale.Kind)
	}
}
