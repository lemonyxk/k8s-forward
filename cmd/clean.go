/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-09 16:28
**/

package cmd

import (
	"context"
	"os"

	"github.com/lemoyxk/console"
	"github.com/lemoyxk/k8s-forward/app"
	"github.com/lemoyxk/k8s-forward/config"
	"github.com/lemoyxk/k8s-forward/dns"
	"github.com/lemoyxk/k8s-forward/ipc"
	"github.com/lemoyxk/k8s-forward/k8s"
	"github.com/lemoyxk/k8s-forward/net"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Clean(record *config.Record) {

	if record == nil {
		return
	}

	if app.RestConfig == nil {
		app.RestConfig = k8s.NewRestConfig()
	}

	if app.Client == nil {
		app.Client = k8s.NewClient()
	}

	// k8s.UnForwardServiceAll()

	ipc.Close()

	net.DeleteNetWork(record)

	dns.DeleteNameServer()

	UnScaleAll(record)

	UnDeploymentAll(record)

	_ = os.RemoveAll(app.Config.RecordPath)
}

func UnScaleAll(record *config.Record) {
	for i := 0; i < len(record.Services); i++ {
		err := UnScale(record.Services[i])
		if err != nil {
			console.Error(err)
		}
	}
}

func UnScale(service *config.Service) error {
	if service == nil {
		return nil
	}

	if service.Switch == nil {
		return nil
	}

	var scale = service.Switch.Scale

	if scale == nil {
		return nil
	}

	if scale.Spec.Replicas == 0 {
		return nil
	}

	var sc, err = GetScale(scale.Kind, scale.Namespace, scale.Name)
	if err != nil {
		return err
	}

	sc.Spec.Replicas = scale.Spec.Replicas

	_, err = UpdateScale(sc, sc.Spec.Replicas)
	if err != nil {
		return err
	}

	console.Warning("recover scale:", sc.Name, "replicas", sc.Spec.Replicas)

	return nil
}

func UnDeploymentAll(record *config.Record) {
	for i := 0; i < len(record.Services); i++ {
		err := UnDeployment(record.Services[i])
		if err != nil {
			console.Error(err)
		}
	}
}

func UnDeployment(service *config.Service) error {
	if service == nil {
		return nil
	}

	if service.Switch == nil {
		return nil
	}

	if service.Switch.Deployment == nil {
		return nil
	}

	var client = app.Client

	var deployment = service.Switch.Deployment

	err := client.AppsV1().Deployments(deployment.Namespace).Delete(context.Background(), deployment.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	console.Warning("delete deployment:", deployment.Name)

	return nil
}
