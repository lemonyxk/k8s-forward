package dns

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/k8s"
	"github.com/lemonyxk/k8s-forward/services"
	utils2 "github.com/lemonyxk/k8s-forward/utils"
	"github.com/lemoyxk/utils"
	"github.com/miekg/dns"
)

var defaultDNS = []string{"8.8.8.8"}

var dnsCache = &cache{}

type cache struct {
	data map[string]*data
	mux  sync.RWMutex
}

type data struct {
	a  []dns.RR
	ip []string
	t  time.Time
}

func (c *cache) init() {
	c.data = make(map[string]*data)
}

func (c *cache) Set(domain string, ip []string, a []dns.RR) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.data[domain] = &data{a: a, ip: ip, t: time.Now()}
}

func (c *cache) Get(domain string) ([]string, []dns.RR) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if d, ok := c.data[domain]; ok {
		if time.Now().Sub(d.t) < time.Minute*1 {
			return d.ip, d.a
		}
	}
	return nil, nil
}

type balance struct {
	mux sync.RWMutex
	dat map[string]int
}

func (b *balance) init() {
	b.dat = make(map[string]int)
}

func (b *balance) Inc(domain string) int {
	b.mux.Lock()
	defer b.mux.Unlock()
	var c = b.dat[domain]
	b.dat[domain] = c + 1
	return c
}

var dnsBalance = &balance{}

type notFound struct {
	mux sync.RWMutex
	dat map[string]bool
}

func (n *notFound) init() {
	n.dat = make(map[string]bool)
}

func (n *notFound) Set(domain string) {
	n.mux.Lock()
	defer n.mux.Unlock()
	n.dat[domain] = true
}

func (n *notFound) Get(domain string) bool {
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.dat[domain]
}

var dnsNotFound = &notFound{}

type handler struct{}

func (t *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	var msg = &dns.Msg{}
	msg.SetReply(r)
	var start = time.Now()
	var domain = msg.Question[0].Name
	var tp = r.Question[0].Qtype
	var class = r.Question[0].Qclass

	if dnsNotFound.Get(domain) {
		err := w.WriteMsg(msg)
		if err != nil {
			console.Error(err)
		}
		return
	}

	// && tp != dns.TypeHTTPS
	if tp != dns.TypeA && tp != dns.TypeAAAA {
		err := w.WriteMsg(msg)
		if err != nil {
			console.Error(err)
		}
		return
	}

	msg.Authoritative = true

	// cache
	var ips, aCache = dnsCache.Get(domain)
	if aCache != nil {
		msg.Answer = aCache
		doHandler(w, domain, ips, msg, start)
		return
	}

	var service *services.Service
	var ok bool
	var headless bool
	var name = domain
	var domainSplit = strings.Split(domain, ".")
	var domainLen = len(domainSplit)
	switch domainLen {
	case 3:
		service = app.Services.Get(domainSplit[1], domainSplit[0])
		ok = service != nil
		if service == nil {
			service = app.Services.Get("default", domainSplit[1])
			ok = service != nil
			if service != nil && service.ClusterIP == "None" {
				ok = false
				service.Pods.Range(func(name string, pod *services.Pod) bool {
					if name == domainSplit[0] {
						ok = true
						headless = true
						return false
					}
					return true
				})
			} else {
				ok = false
			}
		}
	case 4:
		if strings.HasSuffix(domain, ".svc.") {
			service = app.Services.Get(domainSplit[1], domainSplit[0])
			ok = service != nil
		} else {
			service = app.Services.Get(domainSplit[2], domainSplit[1])
			ok = service != nil
			if service != nil && service.ClusterIP == "None" {
				ok = false
				service.Pods.Range(func(name string, pod *services.Pod) bool {
					if name == domainSplit[0] {
						ok = true
						headless = true
						return false
					}
					return true
				})
			} else {
				ok = false
			}
		}
	case 5:
		if strings.HasSuffix(domain, ".svc.") {
			service = app.Services.Get(domainSplit[2], domainSplit[1])
			ok = service != nil
			if service != nil && service.ClusterIP == "None" {
				ok = false
				service.Pods.Range(func(name string, pod *services.Pod) bool {
					if name == domainSplit[0] {
						ok = true
						headless = true
						return false
					}
					return true
				})
			} else {
				ok = false
			}
		}
	case 6:
		if strings.HasSuffix(domain, ".svc.cluster.local.") {
			service = app.Services.Get(domainSplit[1], domainSplit[0])
			ok = service != nil
		}
	case 7:
		if strings.HasSuffix(domain, ".svc.cluster.local.") {
			service = app.Services.Get(domainSplit[2], domainSplit[1])
			ok = service != nil
			if service != nil && service.ClusterIP == "None" {
				ok = false
				service.Pods.Range(func(name string, pod *services.Pod) bool {
					if name == domainSplit[0] {
						ok = true
						headless = true
						return false
					}
					return true
				})
			} else {
				ok = false
			}
		}
	default:
		ok = false
	}

	if ok && service != nil {
		if service.ForwardNumber == 0 && service.Switch == nil {
			var err = k8s.ForwardService(service)
			if err != nil {
				console.Error(err)
			}
		}

		var ips []string

		service.Pods.Range(func(name string, pod *services.Pod) bool {
			if headless {
				if name == domainSplit[0] {
					ips = []string{pod.IP}
					return false
				}
			} else {
				ips = append(ips, pod.IP)
			}

			return true
		})

		if !headless && len(ips) > 1 {
			var index = dnsBalance.Inc(name) % len(ips)
			ips[0], ips[index] = ips[index], ips[0]
		}

		if service.Switch != nil {
			ips = []string{service.Switch.Pod.IP}
		}

		var rr []dns.RR
		var ttl uint32 = 1
		if headless {
			ttl = 300
		}
		for i := 0; i < len(ips); i++ {
			if tp == dns.TypeA {
				var a = &dns.A{
					Hdr: dns.RR_Header{Name: domain, Rrtype: tp, Class: class, Ttl: ttl},
					A:   net.ParseIP(ips[i]),
				}
				rr = append(rr, a)
			} else if tp == dns.TypeAAAA {
				var a = &dns.AAAA{
					Hdr:  dns.RR_Header{Name: domain, Rrtype: tp, Class: class, Ttl: ttl},
					AAAA: net.ParseIP(ips[i]),
				}
				rr = append(rr, a)
			}
		}

		msg.Answer = rr[0:1]

		// if headless {
		// 	dnsCache.Set(domain, ips, rr)
		// }

		doHandler(w, domain, ips, msg, start)
		return
	}

	var err error
	var res *dns.Msg
	var qes = (&dns.Msg{}).SetQuestion(domain, tp)
	for i := 0; i < len(defaultDNS); i++ {
		res, err = dns.Exchange(qes, defaultDNS[i]+":53")
		if err != nil {
			console.Error("trying to exchange", defaultDNS[i], "failed:", err.Error())
		} else {
			break
		}
	}

	if res == nil || res.Rcode != dns.RcodeSuccess || len(res.Answer) == 0 {
		dnsNotFound.Set(domain)
		doHandler(w, domain, ips, msg, start)
		return
	}

	var rr []dns.RR
	for _, i2 := range res.Answer {
		if i2.Header().Rrtype == tp {
			if tp == dns.TypeA {
				ips = []string{i2.(*dns.A).A.String()}
				var a = &dns.A{
					Hdr: dns.RR_Header{Name: domain, Rrtype: tp, Class: class, Ttl: 300},
					A:   net.ParseIP(ips[0]),
				}
				rr = append(rr, a)
			} else if tp == dns.TypeAAAA {
				ips = []string{i2.(*dns.AAAA).AAAA.String()}
				var a = &dns.AAAA{
					Hdr:  dns.RR_Header{Name: domain, Rrtype: tp, Class: class, Ttl: 300},
					AAAA: net.ParseIP(ips[0]),
				}
				rr = append(rr, a)
			}
		}
	}

	msg.Answer = rr
	dnsCache.Set(domain, ips, rr)

	doHandler(w, domain, ips, msg, start)
	return
}

func doHandler(w dns.ResponseWriter, domain string, ip []string, msg *dns.Msg, start time.Time) {
	var i string
	if len(ip) > 0 {
		i = ip[0]
	}
	console.Infof("%s %s %s %s\n", domain[:len(domain)-1],
		dns.TypeToString[msg.Question[0].Qtype], i, time.Since(start))
	err := w.WriteMsg(msg)
	if err != nil {
		console.Error(err)
	}
}

func AddNameServer(svs *services.Services) {
	switch runtime.GOOS {
	case "linux":
		console.Exit("not support linux")
	case "darwin":
		GetDefaultNDS()
		addNameServerDarwin(svs)
	default:
		console.Exit("not support windows")
	}
}

func GetDefaultNDS() {
	f, err := os.OpenFile("/etc/resolv.conf", os.O_RDWR, 0777)
	if err != nil {
		console.Exit(err)
	}

	defer func() { _ = f.Close() }()

	bts, err := io.ReadAll(f)
	if err != nil {
		console.Exit(err)
	}

	var arr []string

	var lines = strings.Split(string(bts), "\n")

	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}

		if lines[i][0] != '#' && strings.Contains(lines[i], "nameserver") {
			arr = append(arr, strings.TrimSpace(strings.ReplaceAll(lines[i], "nameserver", "")))
		}
	}

	if len(arr) != 0 {
		defaultDNS = arr
	}
}

func addNameServerDarwin(svs *services.Services) {

	_ = os.Mkdir("/etc/resolver", 0777)

	var search = []string{
		"default",
		"svc",
		"svc.cluster.local",
	}

	for j := 0; j < len(search); j++ {
		var res = "domain " + search[j] + "\n" + "nameserver " + "127.0.0.1" + "\n" + "port 10053"
		err := utils.File.ReadFromString(res).WriteToPath(`/etc/resolver/` + search[j])
		if err != nil {
			console.Error("dns: domain", search[j], "create failed", err)
		} else {
			// console.Info("dns: domain", search[j], "create success")
		}
	}

	svs.Range(func(name string, service *services.Service) bool {
		var search = []string{
			service.Namespace,
			"svc",
			"svc.cluster.local",
		}

		// if service.Namespace == "default" {
		// 	search = append(search, service.Namespace+".svc")
		// 	search = append(search, service.Namespace+".svc.cluster.local")
		// }

		var model, _ = app.Temp.ReadFile("temp/resolv.conf")
		var res = utils2.ReplaceString(
			string(model),
			[]string{"@domain", "@search", "@ip", "@port"},
			[]string{service.Name, strings.Join(search, " "), "127.0.0.1", "10053"},
		)

		var domain = fmt.Sprintf("%s.svc.cluster.local", name)

		err := utils.File.ReadFromString(res).WriteToPath(`/etc/resolver/` + domain)
		if err != nil {
			console.Error("dns: domain", domain, "create failed", err)
		} else {
			// console.Info("dns: domain", domain, "create success")
		}

		return true
	})
}

func DeleteNameServer(svs *services.Services) {
	switch runtime.GOOS {
	case "linux":
		console.Exit("not support linux")
	case "darwin":
		deleteNameServerDarwin(svs)
	default:
		console.Exit("not support windows")
	}
}

func deleteNameServerDarwin(svs *services.Services) {

	var search = []string{
		"default",
		"svc",
		"svc.cluster.local",
	}

	for j := 0; j < len(search); j++ {
		var err = os.RemoveAll(`/etc/resolver/` + search[j])
		if err != nil {
			console.Error("dns: domain", search[j], "delete failed", err)
		} else {
			// console.Warning("dns: domain", domain, "delete success")
		}
	}

	svs.Range(func(name string, service *services.Service) bool {
		var domain = fmt.Sprintf("%s.svc.cluster.local", name)
		var err = os.RemoveAll(`/etc/resolver/` + domain)
		if err != nil {
			console.Error("dns: domain", domain, "delete failed", err)
		} else {
			// console.Warning("dns: domain", domain, "delete success")
		}
		return true
	})
}

// StartDNS starts a DNS server on the given port.
func StartDNS(fn func()) {
	dnsCache.init()
	dnsBalance.init()
	dnsNotFound.init()
	srv := &dns.Server{Addr: "127.0.0.1:" + strconv.Itoa(10053), Net: "udp"}
	srv.Handler = &handler{}
	srv.NotifyStartedFunc = func() {
		fn()
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			console.Error(err)
		}
	}()
}
