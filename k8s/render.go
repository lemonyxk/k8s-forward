/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-08 17:22
**/

package k8s

import (
	"strings"

	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemoyxk/console"
	"github.com/lemoyxk/utils"
)

func Render() {
	// render
	var table = console.NewTable()

	table.Header("NAMESPACE", "SERVICE_NAME", "POD_NAME", "POD_IP", "POD_PORT")

	for i := 0; i < len(app.Record.Services); i++ {

		if app.Record.Services[i].Pod == nil {
			continue
		}

		table.Row(
			app.Record.Services[i].Namespace,
			app.Record.Services[i].Name,
			app.Record.Services[i].Pod.Name,
			app.Record.Services[i].Pod.IP,
			strings.Join(utils.Extract.Src(app.Record.Services[i].Port).Field("Port").String(), ","),
		)
	}

	console.FgRed.Println(table.Render())
}
