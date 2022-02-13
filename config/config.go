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
	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
)

type Config struct {
	KubePath   string
	RecordPath string
}

type Record struct {
	Services []*Service
	Pods     []*Pod
}

type Status int

const (
	Stop Status = iota
	Start
)

type Service struct {
	Namespace string
	Name      string
	ClusterIP string
	Port      []v1.ServicePort
	Selector  map[string]string
	SelectPod *Pod

	Status   Status
	StopChan chan struct{} `json:"-"`

	Switch *Switch
}

type Switch struct {
	Scale      *v12.Scale
	Deployment *v13.Deployment
	Pod        *Pod
	Status     Status
	StopChan   chan struct{} `json:"-"`
	StopSSH    chan struct{} `json:"-"`
}

type Pod struct {
	Namespace string
	Name      string
	IP        string
	Labels    map[string]string
}
