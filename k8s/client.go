/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-08 10:34
**/

package k8s

import (
	"github.com/lemoyxk/k8s-forward/app"
	"github.com/lemoyxk/k8s-forward/tools"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewRestConfig() *rest.Config {
	// use the current context in kube config
	config, err := clientcmd.BuildConfigFromFlags("", app.Config.KubePath)
	if err != nil {
		tools.Exit(err)
	}

	return config
}

func NewClient() *kubernetes.Clientset {

	// create the client
	client, err := kubernetes.NewForConfig(app.RestConfig)
	if err != nil {
		tools.Exit(err)
	}

	return client
}
