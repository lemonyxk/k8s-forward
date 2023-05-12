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

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemoyxk/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetRecord(namespaces ...string) *config.Record {

	var services []*config.Service

	var pods []*config.Pod

	var client = app.Client

	for i := 0; i < len(namespaces); i++ {
		var namespace = namespaces[i]

		// get services
		svc, err := client.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			console.Exit(err)
		}

		for j := 0; j < len(svc.Items); j++ {
			services = append(services, &config.Service{
				Namespace: svc.Items[j].Namespace,
				Name:      svc.Items[j].Name,
				ClusterIP: svc.Items[j].Spec.ClusterIP,
				Port:      svc.Items[j].Spec.Ports,
				Selector:  svc.Items[j].Spec.Selector,
				Labels:    svc.Items[j].Labels,
				Type:      svc.Items[j].Spec.Type,
			})
		}

		// get pods
		ps, err := client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			console.Exit(err)
		}

		for j := 0; j < len(ps.Items); j++ {
			pods = append(pods, &config.Pod{
				Namespace:   ps.Items[j].Namespace,
				Name:        ps.Items[j].Name,
				IP:          ps.Items[j].Status.PodIP,
				Labels:      ps.Items[j].Labels,
				HostNetwork: ps.Items[j].Spec.HostNetwork,
				Age:         ps.Items[j].CreationTimestamp.Time,
				Restarts:    ps.Items[j].Status.ContainerStatuses[0].RestartCount,
			})
		}
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
				service.Pod = append(service.Pod, pod)
			}
		}
	}

	var record = &config.Record{
		Services:   services,
		Pods:       pods,
		Namespaces: namespaces,
		History:    nil,
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
