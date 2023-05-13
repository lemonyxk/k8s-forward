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
	"sync"
	"sync/atomic"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemonyxk/k8s-forward/tools"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// func ForwardServiceAll() {
// 	for i := 0; i < len(app.Record.Services); i++ {
// 		if app.Record.Services[i].StopForward == nil {
// 			var service = app.Record.Services[i]
// 			_, err := ForwardService(service)
// 			if err != nil {
// 				console.Error(err)
// 				continue
// 			}
// 		}
// 	}
// }
//
// func UnForwardServiceAll() {
// 	for i := 0; i < len(app.Record.Services); i++ {
// 		if app.Record.Services[i].StopForward != nil {
// 			var service = app.Record.Services[i]
// 			service.StopForward <- struct{}{}
// 		}
// 	}
// }

var mux sync.Mutex

func ForwardService(service *config.Service) error {

	mux.Lock()
	defer mux.Unlock()

	if service.ForwardNumber == int32(len(service.Pod)) {
		return nil
	}

	var group = sync.WaitGroup{}

	group.Add(len(service.Pod))

	var client = app.Client

	if len(service.Pod) == 0 {
		return fmt.Errorf("service %s not found", service.Name)
	}

	for i := 0; i < len(service.Pod); i++ {
		var pod = service.Pod[i]

		if pod.Forwarded {
			group.Done()
			continue
		}

		req := client.CoreV1().RESTClient().Post().Namespace(service.Namespace).
			Resource("pods").Name(pod.Name).SubResource(strings.ToLower("PortForward"))

		roundTripper, upgrade, err := spdy.RoundTripperFor(app.RestConfig)
		if err != nil {
			return err
		}

		dialer := spdy.NewDialer(upgrade, &http.Client{Transport: roundTripper}, http.MethodPost, req.URL())

		stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
		out, errOut := new(bytes.Buffer), new(bytes.Buffer)

		var ports = tools.GetServerPorts(service.Port)

		var ip = []string{pod.IP}

		forwarder, err := portforward.NewOnAddresses(dialer, ip, ports, stopChan, readyChan, out, errOut)
		if err != nil {
			return err
		}

		go func() {
			<-readyChan

			pod.Forwarded = true

			atomic.AddInt32(&service.ForwardNumber, 1)

			service.StopForward = append(service.StopForward, stopChan)

			group.Done()

			console.Warning("service forward:", service.Name, pod.IP, ports, "forward start")
		}()

		go func() {
			if err = forwarder.ForwardPorts(); err != nil {
				console.Error(err)
			}

			pod.Forwarded = false

			atomic.AddInt32(&service.ForwardNumber, -1)

			console.Warning("service forward:", service.Name, pod.IP, ports, "forward stop")
		}()
	}

	group.Wait()

	return nil
}

func ForwardPod(namespace string, name string, ip []string, port []string) (chan struct{}, chan struct{}, error) {

	mux.Lock()
	defer mux.Unlock()

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

		console.Warning("pod forward:", name, ip, port, "forward start")
	}()

	go func() {
		if err = forwarder.ForwardPorts(); err != nil {
			console.Error(err)
		}

		console.Warning("pod forward:", name, ip, port, "forward stop")
	}()

	return ready, stopChan, nil
}
