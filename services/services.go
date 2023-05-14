/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2023-05-14 14:03
**/

package services

import (
	"time"

	hash "github.com/lemonyxk/structure/map"
	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
)

type Services struct {
	namespaces []string
	data       *hash.Hash[string, *Service]
	history    []*Pod
}

func NewServices(namespaces ...string) *Services {
	return &Services{namespaces: namespaces, data: hash.New[string, *Service]()}
}

func (s *Services) Namespaces() []string {
	return s.namespaces
}

func (s *Services) Set(namespace, name string, service *Service) {
	s.data.Set(name+"."+namespace, service)
}

func (s *Services) Get(namespace, name string) *Service {
	return s.data.Get(name + "." + namespace)
}

func (s *Services) Range(fn func(name string, service *Service) bool) {
	s.data.Range(func(k string, v *Service) bool {
		return fn(k, v)
	})
}

func (s *Services) Len() int {
	return s.data.Len()
}

func (s *Services) History() []*Pod {
	return s.history
}

func (s *Services) AddHistory(pod *Pod) {
	s.history = append(s.history, pod)
}

type Service struct {
	Namespace string
	Name      string
	ClusterIP string
	Type      v1.ServiceType
	Port      []v1.ServicePort
	Selector  map[string]string
	Labels    map[string]string
	Pods      *Pods

	ForwardNumber int32

	Switch *Switch
}

type Switch struct {
	Scale      *v12.Scale
	Deployment *v13.Deployment
	Pod        *Pod

	StopForward chan struct{} `json:"-"`
	StopSSH     chan struct{} `json:"-"`
}

type Pod struct {
	Namespace   string
	Name        string
	IP          string
	Labels      map[string]string
	HostNetwork bool
	Age         time.Time
	Restarts    int32
	Phase       v1.PodPhase
	Containers  []v1.Container

	Forwarded   bool
	StopForward chan struct{} `json:"-"`
}
