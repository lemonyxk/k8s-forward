/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-08 10:35
**/

package config

import (
	"time"

	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
)

type Config struct {
	HomePath   string
	KubePath   string
	RecordPath string
}

type Record struct {
	Services   []*Service
	Pods       []*Pod
	Namespaces []string
	History    []*Pod
}

type Service struct {
	Namespace string
	Name      string
	ClusterIP string
	Type      v1.ServiceType
	Port      []v1.ServicePort
	Selector  map[string]string
	Labels    map[string]string
	Pod       []*Pod

	ForwardNumber int32
	StopForward   []chan struct{} `json:"-"`

	Switch *Switch
}

type Status int

const (
	Stop Status = iota
	Start
)

type Switch struct {
	Scale       *v12.Scale
	Deployment  *v13.Deployment
	Pod         *Pod
	Status      Status
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
}
