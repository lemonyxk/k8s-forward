/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-08 10:43
**/

package app

import (
	"embed"

	"github.com/lemoyxk/k8s-forward/config"
	"github.com/lemoyxk/k8s-forward/manager"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//go:embed temp
var Temp embed.FS

var Config *config.Config

var Record *config.Record

var RestConfig *rest.Config

var DnsDomain = make(map[string]*config.Service)

var Client *kubernetes.Clientset

var Manager *manager.Manager
