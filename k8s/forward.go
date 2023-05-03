/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-08 19:49
**/

package k8s

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemonyxk/k8s-forward/tools"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func ForwardServiceAll() {
	for i := 0; i < len(app.Record.Services); i++ {
		if app.Record.Services[i].StopForward == nil {
			var service = app.Record.Services[i]
			_, err := ForwardService(service)
			if err != nil {
				console.Error(err)
				continue
			}
		}
	}
}

func UnForwardServiceAll() {
	for i := 0; i < len(app.Record.Services); i++ {
		if app.Record.Services[i].StopForward != nil {
			var service = app.Record.Services[i]
			service.StopForward <- struct{}{}
		}
	}
}

func ForwardService(service *config.Service) (chan struct{}, error) {

	var ch = make(chan struct{}, 1)

	var client = app.Client

	if service.Pod == nil {
		return nil, fmt.Errorf("service %s not found", service.Name)
	}

	var pod = service.Pod

	req := client.CoreV1().RESTClient().Post().Namespace(service.Namespace).
		Resource("pods").Name(pod.Name).SubResource(strings.ToLower("PortForward"))

	roundTripper, upgrade, err := spdy.RoundTripperFor(app.RestConfig)
	if err != nil {
		return nil, err
	}

	dialer := spdy.NewDialer(upgrade, &http.Client{Transport: roundTripper}, http.MethodPost, req.URL())

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	var ports = tools.GetServerPorts(service.Port)

	var ip = []string{pod.IP}

	forwarder, err := portforward.NewOnAddresses(dialer, ip, ports, stopChan, readyChan, out, errOut)
	if err != nil {
		return nil, err
	}

	go func() {
		<-readyChan

		ch <- struct{}{}

		service.Status = config.Start
		service.StopForward = stopChan

		console.Info("service forward:", service.Name, pod.IP, ports, "forward start")
	}()

	go func() {
		if err = forwarder.ForwardPorts(); err != nil {
			console.Error(err)
		}

		console.Warning("service forward:", service.Name, pod.IP, ports, "forward stop")

		service.Status = config.Stop
	}()

	return ch, nil
}

func ForwardPod(namespace string, name string, ip []string, port []string) (chan struct{}, chan struct{}, error) {

	var ready = make(chan struct{}, 1)

	var client = app.Client

	req := client.CoreV1().RESTClient().Post().Namespace(namespace).
		Resource("pods").Name(name).SubResource(strings.ToLower("PortForward"))

	roundTripper, upgrade, err := spdy.RoundTripperFor(app.RestConfig)
	if err != nil {
		return nil, nil, err
	}

	dialer := spdy.NewDialer(upgrade, &http.Client{Transport: roundTripper}, http.MethodPost, req.URL())

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	forwarder, err := portforward.NewOnAddresses(dialer, ip, port, stopChan, readyChan, out, errOut)
	if err != nil {
		return nil, nil, err
	}

	go func() {
		<-readyChan

		ready <- struct{}{}

		console.Info("pod forward:", name, ip, port, "forward start")
	}()

	go func() {
		if err = forwarder.ForwardPorts(); err != nil {
			console.Error(err)
		}

		console.Warning("pod forward:", name, ip, port, "forward stop")

	}()

	return ready, stopChan, nil
}
