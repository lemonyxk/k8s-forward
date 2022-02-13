package dns

import (
	"github.com/lemoyxk/k8s-forward/app"
	"github.com/lemoyxk/k8s-forward/config"
)

func GetDNSDomain() map[string]*config.Service {

	var dnsDomain = make(map[string]*config.Service)

	for i := 0; i < len(app.Record.Services); i++ {
		if app.Record.Services[i].Pod == nil {
			continue
		}

		var namespace = "." + app.Record.Services[i].Namespace

		dnsDomain[app.Record.Services[i].Name+namespace+"."] = app.Record.Services[i]

		// special for default namespace
		if namespace == ".default" {
			namespace = ""
			dnsDomain[app.Record.Services[i].Name+namespace+"."] = app.Record.Services[i]
		}

	}

	return dnsDomain
}
