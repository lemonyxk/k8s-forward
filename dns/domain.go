package dns

import (
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
)

func GetDNSDomain() map[string]*config.Service {

	var dnsDomain = make(map[string]*config.Service)

	for i := 0; i < len(app.Record.Services); i++ {
		if len(app.Record.Services[i].Pod) == 0 {
			continue
		}

		var namespace = app.Record.Services[i].Namespace
		//
		// dnsDomain[app.Record.Services[i].Name+namespace+"."] = app.Record.Services[i]
		//
		// // special for default namespace
		// if namespace == ".default" {
		// 	namespace = ""
		// 	dnsDomain[app.Record.Services[i].Name+namespace+"."] = app.Record.Services[i]
		// }

		var domain = app.Record.Services[i].Name + "." + namespace + "."

		dnsDomain[domain] = app.Record.Services[i]
	}

	return dnsDomain
}
