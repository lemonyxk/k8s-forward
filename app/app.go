/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-08 10:43
**/

package app

import (
	"embed"

	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemonyxk/k8s-forward/services"
	"github.com/lemonyxk/k8s-forward/utils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//go:embed temp
var Temp embed.FS

var SSHForwardKey = utils.RandomString(12)

var Config *config.Config

var RestConfig *rest.Config

var Services *services.Services

var Client *kubernetes.Clientset

var Watch *Watcher
