/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-08 11:12
**/

package app

import (
	"context"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/services"
	utils2 "github.com/lemonyxk/k8s-forward/utils"
	"github.com/lemoyxk/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Load(namespaces ...string) {

	var client = Client

	for i := 0; i < len(namespaces); i++ {
		var namespace = namespaces[i]

		// get services
		svc, err := client.CoreV1().Services(namespace).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			console.Exit(err)
		}

		for j := 0; j < len(svc.Items); j++ {
			var service = &services.Service{
				Namespace: svc.Items[j].Namespace,
				Name:      svc.Items[j].Name,
				ClusterIP: svc.Items[j].Spec.ClusterIP,
				Port:      svc.Items[j].Spec.Ports,
				Selector:  svc.Items[j].Spec.Selector,
				Labels:    svc.Items[j].Labels,
				Type:      svc.Items[j].Spec.Type,
				Pods:      services.NewPods(),
			}

			Services.Set(namespace, svc.Items[j].Name, service)
		}

		// get pods
		ps, err := client.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			console.Exit(err)
		}

		for j := 0; j < len(ps.Items); j++ {
			var pod = &services.Pod{
				Namespace:   ps.Items[j].Namespace,
				Name:        ps.Items[j].Name,
				IP:          ps.Items[j].Status.PodIP,
				Labels:      ps.Items[j].Labels,
				HostNetwork: ps.Items[j].Spec.HostNetwork,
				Age:         ps.Items[j].CreationTimestamp.Time,
				Restarts:    ps.Items[j].Status.ContainerStatuses[0].RestartCount,
				Phase:       ps.Items[j].Status.Phase,
				Containers:  ps.Items[j].Spec.Containers,
				Forwarded:   false,
				StopForward: nil,
			}

			Services.AddHistory(pod)

			Services.Range(func(name string, service *services.Service) bool {
				if pod.Namespace != service.Namespace {
					return true
				}

				if utils2.Match(pod.Labels, service.Selector) {
					service.Pods.Set(pod.Name, pod)
				}

				return true
			})
		}
	}

	SaveAllServices(Services)

	console.Info("load services success", Services.Len())
}

func SaveAllServices(svs *services.Services) {

	SaveServices(svs)

	SavePods(svs)

	SaveHistory(svs)

	SaveNamespaces(svs)
}

func SaveNamespaces(svs *services.Services) {
	// namespace.json
	var namespacePath = filepath.Join(Config.HomePath, "namespaces.json")
	var namespaceList = svs.Namespaces()

	bts, err := jsoniter.Marshal(namespaceList)
	if err != nil {
		console.Exit(err)
	}

	err = utils.File.ReadFromBytes(bts).WriteToPath(namespacePath)
	if err != nil {
		console.Exit(err)
	}
}

func SaveServices(svs *services.Services) {
	// services.json
	var svcPath = filepath.Join(Config.HomePath, "services.json")
	var serviceList []*services.Service

	svs.Range(func(name string, service *services.Service) bool {
		serviceList = append(serviceList, service)
		return true
	})

	bts, err := jsoniter.Marshal(serviceList)
	if err != nil {
		console.Exit(err)
	}

	err = utils.File.ReadFromBytes(bts).WriteToPath(svcPath)
	if err != nil {
		console.Exit(err)
	}
}

func SavePods(svs *services.Services) {
	// pods.json
	var podPath = filepath.Join(Config.HomePath, "pods.json")
	var podList []*services.Pod

	svs.Range(func(name string, service *services.Service) bool {
		service.Pods.Range(func(name string, pod *services.Pod) bool {
			podList = append(podList, pod)
			return true
		})
		return true
	})

	bts, err := jsoniter.Marshal(podList)
	if err != nil {
		console.Exit(err)
	}

	err = utils.File.ReadFromBytes(bts).WriteToPath(podPath)
	if err != nil {
		console.Exit(err)
	}
}

func SaveHistory(svs *services.Services) {
	// history.json
	var historyPath = filepath.Join(Config.HomePath, "history.json")
	var historyList = svs.History()

	bts, err := jsoniter.Marshal(historyList)
	if err != nil {
		console.Exit(err)
	}

	err = utils.File.ReadFromBytes(bts).WriteToPath(historyPath)
	if err != nil {
		console.Exit(err)
	}
}

func LoadAllServices() *services.Services {

	var namespaces []string

	// namespace.json
	var namespacePath = filepath.Join(Config.HomePath, "namespaces.json")
	var res = utils.File.ReadFromPath(namespacePath)
	if res.LastError() != nil {
		return nil
	}

	var err = utils.Json.Decode(res.Bytes(), &namespaces)
	if err != nil {
		return nil
	}

	var svs = services.NewServices()

	// services.json
	var svcPath = filepath.Join(Config.HomePath, "services.json")
	res = utils.File.ReadFromPath(svcPath)
	if res.LastError() != nil {
		return nil
	}

	var serviceList []*services.Service
	err = utils.Json.Decode(res.Bytes(), &serviceList)
	if err != nil {
		return nil
	}

	for i := 0; i < len(serviceList); i++ {
		svs.Set(serviceList[i].Namespace, serviceList[i].Name, serviceList[i])
	}

	// pods.json
	var podPath = filepath.Join(Config.HomePath, "pods.json")
	res = utils.File.ReadFromPath(podPath)
	if res.LastError() != nil {
		return nil
	}

	var podList []*services.Pod
	err = utils.Json.Decode(res.Bytes(), &podList)
	if err != nil {
		return nil
	}

	for i := 0; i < len(podList); i++ {
		svs.Range(func(name string, service *services.Service) bool {
			if podList[i].Namespace != service.Namespace {
				return true
			}

			if utils2.Match(podList[i].Labels, service.Selector) {
				service.Pods.Set(podList[i].Name, podList[i])
			}

			return true
		})
	}

	// history.json
	var historyPath = filepath.Join(Config.HomePath, "history.json")
	res = utils.File.ReadFromPath(historyPath)
	if res.LastError() != nil {
		return nil
	}

	var historyList []*services.Pod
	err = utils.Json.Decode(res.Bytes(), &historyList)
	if err != nil {
		return nil
	}

	for i := 0; i < len(historyList); i++ {
		svs.AddHistory(historyList[i])
	}

	return svs
}
