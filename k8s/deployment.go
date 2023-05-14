/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2023-05-13 23:23
**/

package k8s

import (
	"context"
	"io"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/services"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func UnSwitchDeploymentAll(svs *services.Services) {
	svs.Range(func(name string, service *services.Service) bool {
		err := UnSwitchDeployment(service)
		if err != nil {
			console.Error(err)
		}
		return true
	})
}

func UnSwitchDeployment(service *services.Service) error {
	if service == nil {
		return nil
	}

	if service.Switch == nil {
		return nil
	}

	if service.Switch.Deployment == nil {
		return nil
	}

	var deployment = service.Switch.Deployment

	err := UnDeployment(deployment)
	if err != nil {
		return err
	}

	return nil
}

func Deployment(deployment *v12.Deployment) (*v12.Deployment, error) {

	var client = app.Client

	res, err := client.AppsV1().Deployments(deployment.Namespace).Create(context.Background(), deployment, v1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	console.Warning("deployment", deployment.Namespace, deployment.Name, "create success")

	return res, nil
}

func UnDeployment(deployment *v12.Deployment) error {

	var client = app.Client

	err := client.AppsV1().Deployments(deployment.Namespace).Delete(context.Background(), deployment.Name, v1.DeleteOptions{})
	if err != nil {
		return err
	}

	console.Warning("deployment", deployment.Namespace, deployment.Name, "delete success")

	return nil
}

func GenerateDeployment() (*v12.Deployment, error) {
	var f, err = app.Temp.Open("temp/deployment.yaml")
	if err != nil {
		return nil, err
	}

	bts, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var deployment v12.Deployment

	err = yaml.Unmarshal(bts, &deployment)
	if err != nil {
		return nil, err
	}

	return &deployment, nil
}
