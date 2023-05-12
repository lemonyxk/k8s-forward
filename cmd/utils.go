/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2023-05-12 15:56
**/

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	v12 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetScale(resource string, namespace string, name string) (*v12.Scale, error) {

	var client = app.Client

	resource = strings.ToLower(resource)

	var scale *v12.Scale

	switch resource {
	case "deployment":
		s, err := client.AppsV1().
			Deployments(namespace).
			GetScale(context.Background(), name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}

		scale = s

	case "statefulset":
		s, err := client.AppsV1().StatefulSets(namespace).
			GetScale(context.Background(), name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}

		scale = s

	case "daemonset":
		return nil, fmt.Errorf("daemonset not support")
	case "replicaset":
		s, err := client.AppsV1().ReplicaSets(namespace).
			GetScale(context.Background(), name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}

		scale = s

	default:
		return nil, fmt.Errorf("%s not support", resource)
	}

	scale.Kind = resource

	return scale, nil
}

func UpdateScale(scale *v12.Scale, replicas int32) (*v12.Scale, error) {

	var client = app.Client

	sc := *scale
	sc.Spec.Replicas = replicas

	console.Warning("update scale:", scale.Name, "replicas:", replicas)

	switch scale.Kind {
	case "deployment":
		return client.AppsV1().Deployments(scale.Namespace).UpdateScale(context.TODO(), scale.Name, &sc, v1.UpdateOptions{})
	case "statefulset":
		return client.AppsV1().StatefulSets(scale.Namespace).UpdateScale(context.TODO(), scale.Name, &sc, v1.UpdateOptions{})
	case "daemonset":
		return nil, fmt.Errorf("daemonset not support")
	case "replicaset":
		return client.AppsV1().ReplicaSets(scale.Namespace).UpdateScale(context.TODO(), scale.Name, &sc, v1.UpdateOptions{})
	default:
		return nil, fmt.Errorf("%s not support", scale.Kind)
	}
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

	err := client.AppsV1().Deployments(deployment.Namespace).Delete(context.Background(), deployment.Name, v1.DeleteOptions{})
	if err != nil {
		return err
	}

	console.Warning("delete deployment:", deployment.Name)

	return nil
}
