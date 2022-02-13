/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-12 22:55
**/

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/lemoyxk/console"
	"github.com/lemoyxk/k8s-forward/app"
	"github.com/lemoyxk/k8s-forward/config"
	"github.com/lemoyxk/k8s-forward/k8s"
	"github.com/lemoyxk/k8s-forward/net"
	"github.com/lemoyxk/k8s-forward/tools"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func Recover(args []string) string {
	var resource = tools.GetArgs([]string{"recover", "-r", "--recover"}, args)
	if resource == "" {
		return "recover resource is empty"
	}

	var name = tools.GetArgs([]string{resource}, args)
	if name == "" {
		return "recover resource name is empty"
	}

	var namespace = tools.GetArgs([]string{"--namespace", "-n"}, args)
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

	console.Info("match service:", service.Name, "replicas:", scale.Spec.Replicas)

	err = UnScale(service)
	if err != nil {
		return err.Error()
	}

	console.Info("update service:", service.Name, "replicas:", service.Switch.Scale.Spec.Replicas)

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
	var pod *v12.Pod

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
						pod = p
						watch.Stop()
					}
				}
			}
		}
	}()

	<-ch

	err = UnDeployment(service)
	if err != nil {
		return err.Error()
	}

	service.SelectPod = &config.Pod{Namespace: pod.Namespace, Name: pod.Name, IP: pod.Status.PodIP, Labels: pod.Labels}
	k8s.SaveRecordToFile(app.Record)

	net.CreateNetWorkByIp(service.SelectPod.IP)

	// delete switch pod ip
	net.DeleteNetWorkByIp(service.Switch.Pod.IP)

	service.Switch.Deployment = nil
	k8s.SaveRecordToFile(app.Record)

	service.Switch.Pod = nil
	k8s.SaveRecordToFile(app.Record)

	service.Switch.StopChan <- struct{}{}
	service.Switch.StopSSH <- struct{}{}

	service.Switch = nil
	k8s.SaveRecordToFile(app.Record)

	return fmt.Sprintf("%s %s %s recover success", resource, namespace, name)
}
