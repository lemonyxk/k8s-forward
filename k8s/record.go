/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-08 11:12
**/

package k8s

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/lemoyxk/console"
	"github.com/lemoyxk/k8s-forward/app"
	"github.com/lemoyxk/k8s-forward/config"
	"github.com/lemoyxk/k8s-forward/tools"
	"github.com/lemoyxk/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetRecord() *config.Record {
	var client = app.Client

	// get services
	svc, err := client.CoreV1().Services("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		tools.Exit(err)
	}

	var services []*config.Service
	for i := 0; i < len(svc.Items); i++ {

		// var selector = make(map[string]string)
		//
		// for k, v := range svc.Items[i].Spec.Selector {
		// 	selector[k] = v
		// }

		// for k, v := range svc.Items[i].Labels {
		// 	labels[k] = v
		// }

		services = append(services, &config.Service{
			Namespace: svc.Items[i].Namespace,
			Name:      svc.Items[i].Name,
			ClusterIP: svc.Items[i].Spec.ClusterIP,
			Port:      svc.Items[i].Spec.Ports,
			Selector:  svc.Items[i].Spec.Selector,
		})
	}

	// get pods
	ps, err := client.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		tools.Exit(err)
	}

	var pods []*config.Pod
	for i := 0; i < len(ps.Items); i++ {
		pods = append(pods, &config.Pod{
			Namespace:   ps.Items[i].Namespace,
			Name:        ps.Items[i].Name,
			IP:          ps.Items[i].Status.PodIP,
			Labels:      ps.Items[i].Labels,
			HostNetwork: ps.Items[i].Spec.HostNetwork,
		})
	}

	// Match pods to services
	for i := 0; i < len(services); i++ {
		var service = services[i]

		for j := 0; j < len(pods); j++ {
			var pod = pods[j]

			if pod.Namespace != service.Namespace {
				continue
			}

			if Match(pod.Labels, service.Selector) {
				service.Pod = pod
				break
			}
		}
	}

	var record = &config.Record{
		Services: services,
		Pods:     pods,
	}

	return record
}

func SaveRecordToFile(record *config.Record) {
	_ = os.MkdirAll(filepath.Dir(app.Config.RecordPath), 0755)

	var err = utils.File.ReadFromBytes(utils.Json.Encode(record)).WriteToPath(app.Config.RecordPath)
	if err != nil {
		console.Error(err)
	}
}

func GetRecordFromFile() *config.Record {
	_ = os.MkdirAll(filepath.Dir(app.Config.RecordPath), 0755)

	var res = utils.File.ReadFromPath(app.Config.RecordPath)
	if res.LastError() != nil {
		return nil
	}

	var record config.Record

	var err = utils.Json.Decode(res.Bytes(), &record)
	if err != nil {
		return nil
	}

	return &record
}

func Match(labels map[string]string, selector map[string]string) bool {
	for k1, v1 := range labels {
		if selector[k1] == v1 {
			return true
		}
	}
	return false
}

func MakeLabels(str string) map[string]string {

	var res = make(map[string]string)
	var arr = strings.Split(str, ",")
	for i := 0; i < len(arr); i++ {
		var v = strings.Split(arr[i], "=")
		if len(v) != 2 {
			continue
		}
		res[v[0]] = v[1]
	}

	return res
}

func MakeLabelsString(labels map[string]string) string {
	var res = ""
	for k, v := range labels {
		res += k + "=" + v + ","
	}
	return strings.TrimRight(res, ",")
}
