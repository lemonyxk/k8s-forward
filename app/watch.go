/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-08 17:15
**/

package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/services"
	"github.com/lemonyxk/k8s-forward/utils"
	hash "github.com/lemonyxk/structure/map"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch2 "k8s.io/apimachinery/pkg/watch"
)

func NewWatcher(namespaces ...string) *Watcher {
	return &Watcher{
		Namespaces: namespaces,
		UpdateList: hash.New[string, *v12.Pod](),
		StartTime:  time.Now(),
	}
}

type Watcher struct {
	Watches    []watch2.Interface
	StartTime  time.Time
	Listen     []*Filter
	Namespaces []string
	UpdateList *hash.Hash[string, *v12.Pod]
}

type Filter struct {
	Namespace string
	Selector  map[string]string
	Name      string
	Number    int32

	pods []*v12.Pod
	ch   chan []*v12.Pod
}

func (w *Watcher) Watch(filter *Filter) chan []*v12.Pod {
	filter.ch = make(chan []*v12.Pod)
	w.Listen = append(w.Listen, filter)
	return filter.ch
}

func (w *Watcher) Stop() {
	for i := 0; i < len(w.Watches); i++ {
		w.Watches[i].Stop()
	}
}

func (w *Watcher) Run() {

	for i := 0; i < len(w.Namespaces); i++ {

		var namespace = w.Namespaces[i]

		console.Info("watch namespace:", namespace)

		// factory := informers.NewSharedInformerFactoryWithOptions(Client, 0, informers.WithNamespace(namespace))
		// informer := factory.Core().V1().Pods().Informer()
		// _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// 	AddFunc: func(obj interface{}) {
		// 		pod, ok := obj.(*v12.Pod)
		// 		if !ok {
		// 			return
		// 		}
		// 		w.OnAdd(pod)
		// 	},
		// 	DeleteFunc: func(obj interface{}) {
		// 		pod, ok := obj.(*v12.Pod)
		// 		if !ok {
		// 			return
		// 		}
		// 		w.OnDelete(pod)
		// 	},
		// 	UpdateFunc: func(oldObj, newObj interface{}) {
		// 		pod, ok := newObj.(*v12.Pod)
		// 		if !ok {
		// 			return
		// 		}
		// 		w.OnUpdate(pod)
		// 	},
		// })
		// if err != nil {
		// 	console.Error(err)
		// }
		//
		// err = informer.SetWatchErrorHandler(func(r *cache.Reflector, err error) {
		// 	console.Error(err)
		// })
		// if err != nil {
		// 	console.Error(err)
		// }
		//
		// factory.Start(wait.NeverStop)

		watch, err := Client.CoreV1().Pods(namespace).Watch(context.Background(), v1.ListOptions{
			Watch: true,
			// LabelSelector: labels.Set(service.Selector).AsSelector().String(),
		})
		if err != nil {
			console.Exit(err)
		}

		w.Watches = append(w.Watches, watch)

		go func() {
			for {
				select {
				case event, ok := <-watch.ResultChan():
					if !ok {
						console.Error("lose connection with k8s")
						return
					}

					pod, ok := event.Object.(*v12.Pod)
					if !ok {
						continue
					}

					switch event.Type {
					case watch2.Deleted:
						go w.OnDelete(pod)
					case watch2.Added:
						go w.OnAdd(pod)
					case watch2.Modified:
						go w.OnUpdate(pod)
					case watch2.Error:
						console.Error("watch error:", event.Type, pod.Namespace, pod.Name, pod.Status.Phase)
					default:

					}
				}
			}
		}()
	}
}

func (w *Watcher) OnAdd(pod *v12.Pod) {

	var name = fmt.Sprintf("%s.%s", pod.Name, pod.Namespace)

	if pod.CreationTimestamp.Time.Sub(w.StartTime) < 0 {
		return
	}

	for {
		pod = w.UpdateList.Get(name)
		if pod != nil && pod.Status.Phase == v12.PodRunning {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	var p = &services.Pod{
		Namespace:   pod.Namespace,
		Name:        pod.Name,
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

	for i := 0; i < len(w.Listen); i++ {
		var l = w.Listen[i]
		if !utils.Match(pod.Labels, l.Selector) {
			continue
		}

		if !strings.HasPrefix(pod.Name, l.Name) {
			continue
		}

		l.pods = append(l.pods, pod)

		if len(l.pods) != int(l.Number) {
			continue
		}

		w.Listen = append(w.Listen[:i], w.Listen[i+1:]...)
		go func() {
			l.ch <- l.pods
		}()
	}

	Services.Range(func(name string, service *services.Service) bool {

		if service.Selector == nil {
			return true
		}

		if !utils.Match(pod.Labels, service.Selector) {
			return true
		}

		service.Pods.Set(pod.Name, p)

		CreateNetWorkByPod(pod)

		Services.AddHistory(p)

		SaveHistory(Services)

		console.Info("new pod:", pod.Namespace, pod.Name, "ip:", pod.Status.PodIP)

		return false
	})
}

func (w *Watcher) OnUpdate(pod *v12.Pod) {
	var name = fmt.Sprintf("%s.%s", pod.Name, pod.Namespace)
	w.UpdateList.Set(name, pod)
}

func (w *Watcher) OnDelete(pod *v12.Pod) {
	Services.Range(func(name string, service *services.Service) bool {

		if service.Selector == nil {
			return true
		}

		if !utils.Match(pod.Labels, service.Selector) {
			return true
		}

		var p = service.Pods.Get(pod.Name)
		if p == nil {
			return true
		}

		if p.Forwarded {
			p.StopForward <- struct{}{}
			p.Forwarded = false
		}

		service.Pods.Delete(pod.Name)

		console.Info("delete pod:", pod.Namespace, pod.Name, "ip:", pod.Status.PodIP)

		return false
	})
}
