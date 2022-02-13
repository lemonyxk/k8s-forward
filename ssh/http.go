/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-13 11:30
**/

package ssh

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/lemoyxk/console"
)

func Http(scheme, addr string, list []string) {

	if len(list) == 0 {
		console.Error("[-] no target")
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if len(list) == 1 {
			r.Host = list[0]
		}

		if len(list) == 2 {
			r.Host = strings.ReplaceAll(r.Host, list[0], list[1])
		}

		httputil.
			NewSingleHostReverseProxy(&url.URL{Scheme: scheme, Host: r.Host}).
			ServeHTTP(w, r)
	})

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		console.Error(err)
	}
}
