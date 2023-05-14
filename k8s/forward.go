/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
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
	"github.com/lemonyxk/k8s-forward/services"
	"github.com/lemonyxk/k8s-forward/utils"
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

func ForwardService(service *services.Service) error {

	mux.Lock()
	defer mux.Unlock()

	if service.Pods.Len() == 0 {
		return fmt.Errorf("service %s not found", service.Name)
	}

	if service.ForwardNumber == int32(service.Pods.Len()) {
		return nil
	}

	var group = sync.WaitGroup{}

	group.Add(service.Pods.Len())

	var client = app.Client

	service.Pods.Range(func(name string, pod *services.Pod) bool {
		if pod.Forwarded {
			group.Done()
			return true
		}

		req := client.CoreV1().RESTClient().Post().Namespace(service.Namespace).
			Resource("pods").Name(pod.Name).SubResource(strings.ToLower("PortForward"))

		roundTripper, upgrade, err := spdy.RoundTripperFor(app.RestConfig)
		if err != nil {
			return false
		}

		dialer := spdy.NewDialer(upgrade, &http.Client{Transport: roundTripper}, http.MethodPost, req.URL())

		stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
		out, errOut := new(bytes.Buffer), new(bytes.Buffer)

		var ports = utils.GetServerPorts(service.Port)

		var ip = []string{pod.IP}

		forwarder, err := portforward.NewOnAddresses(dialer, ip, ports, stopChan, readyChan, out, errOut)
		if err != nil {
			console.Error(err)
			return false
		}

		go func() {
			err = forwarder.ForwardPorts()
			if err != nil {
				console.Error(err)
			}
			pod.Forwarded = false
			pod.StopForward = nil
			atomic.AddInt32(&service.ForwardNumber, -1)
			console.Warning("service forward:", service.Namespace, service.Name, pod.IP, ports, "forward stop")
		}()

		go func() {
			<-readyChan
			pod.Forwarded = true
			pod.StopForward = stopChan
			atomic.AddInt32(&service.ForwardNumber, 1)
			console.Warning("service forward:", service.Namespace, service.Name, pod.IP, ports, "forward start")
			group.Done()
		}()

		return true
	})

	group.Wait()

	return nil
}

func ForwardPod(pod *services.Pod, ip []string, port []string) (chan struct{}, chan struct{}, error) {

	mux.Lock()
	defer mux.Unlock()

	var ready = make(chan struct{}, 1)

	var client = app.Client

	req := client.CoreV1().RESTClient().Post().Namespace(pod.Namespace).
		Resource("pods").Name(pod.Name).SubResource(strings.ToLower("PortForward"))

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

		pod.Forwarded = true

		pod.StopForward = stopChan

		ready <- struct{}{}

		console.Warning("pod forward:", pod.Namespace, pod.Name, ip, port, "forward start")
	}()

	go func() {
		if err = forwarder.ForwardPorts(); err != nil {
			console.Error(err)
		}

		pod.Forwarded = false

		pod.StopForward = nil

		console.Warning("pod forward:", pod.Namespace, pod.Name, ip, port, "forward stop")
	}()

	return ready, stopChan, nil
}
