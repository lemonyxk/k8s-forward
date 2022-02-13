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
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/lemoyxk/console"
)

type Handler struct {
	Scheme string
	List   []string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(h.List) == 1 {
		r.Host = h.List[0]
	}

	if len(h.List) == 2 {
		r.Host = strings.ReplaceAll(r.Host, h.List[0], h.List[1])
	}

	httputil.
		NewSingleHostReverseProxy(&url.URL{Scheme: h.Scheme, Host: r.Host}).
		ServeHTTP(w, r)
}

func Http(l net.Listener, scheme string, list []string) {
	if len(list) == 0 {
		console.Error("[-] no target")
		return
	}

	var handler = &Handler{Scheme: scheme, List: list}

	var server = http.Server{Handler: handler}

	err := server.Serve(l)
	if err != nil {
		console.Error(err)
		return
	}
}
