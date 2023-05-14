/**
* @program: k8s-forward
*
* @description:
*
* @author: lemon
*
* @create: 2022-02-09 11:40
**/

package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/services"
	v12 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetScale(resource string, namespace string, name string) (*v12.Scale, error) {

	var client = app.Client

	resource = strings.ToLower(resource)

	var scale *v12.Scale
	var err error

	switch resource {
	case "deployment":
		scale, err = client.AppsV1().Deployments(namespace).GetScale(context.Background(), name, v1.GetOptions{})
	case "statefulset":
		scale, err = client.AppsV1().StatefulSets(namespace).GetScale(context.Background(), name, v1.GetOptions{})
	case "replicaset":
		scale, err = client.AppsV1().ReplicaSets(namespace).GetScale(context.Background(), name, v1.GetOptions{})
	case "daemonset":
		scale, err = nil, fmt.Errorf("daemonset not support")
	default:
		scale, err = nil, fmt.Errorf("%s not support", resource)
	}

	if scale != nil {
		scale.Kind = resource
	}

	return scale, err
}

func Scale(scale *v12.Scale, replicas int32) (*v12.Scale, error) {

	var client = app.Client

	var copyScale = scale.DeepCopy()

	copyScale.Spec.Replicas = replicas

	var res *v12.Scale
	var err error

	switch copyScale.Kind {
	case "deployment":
		res, err = client.AppsV1().Deployments(copyScale.Namespace).UpdateScale(context.TODO(), copyScale.Name, copyScale, v1.UpdateOptions{})
	case "statefulset":
		res, err = client.AppsV1().StatefulSets(copyScale.Namespace).UpdateScale(context.TODO(), copyScale.Name, copyScale, v1.UpdateOptions{})
	case "replicaset":
		res, err = client.AppsV1().ReplicaSets(copyScale.Namespace).UpdateScale(context.TODO(), copyScale.Name, copyScale, v1.UpdateOptions{})
	case "daemonset":
		res, err = nil, fmt.Errorf("daemonset not support")
	default:
		res, err = nil, fmt.Errorf("%s not support", copyScale.Kind)
	}

	if res != nil {
		res.Kind = copyScale.Kind
	}

	console.Warning("scale", scale.Namespace, scale.Name, "replicas", scale.Spec.Replicas, "to", replicas, "success")

	return res, err
}

func UnSwitchScaleAll(svs *services.Services) {
	svs.Range(func(name string, service *services.Service) bool {
		err := UnSwitchScale(service)
		if err != nil {
			console.Error(err)
		}
		return true
	})
}

func UnSwitchScale(service *services.Service) error {
	if service == nil {
		return nil
	}

	if service.Switch == nil {
		return nil
	}

	if service.Switch.StopForward != nil {
		service.Switch.StopForward <- struct{}{}
	}

	if service.Switch.StopSSH != nil {
		service.Switch.StopSSH <- struct{}{}
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

	_, err = Scale(sc, scale.Spec.Replicas)
	if err != nil {
		return err
	}

	return nil
}
