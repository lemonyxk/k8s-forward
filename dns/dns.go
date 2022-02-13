package dns

import (
	"bufio"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lemoyxk/console"
	"github.com/lemoyxk/k8s-forward/app"
	"github.com/lemoyxk/k8s-forward/config"
	"github.com/lemoyxk/k8s-forward/k8s"
	"github.com/lemoyxk/k8s-forward/tools"
	"github.com/lemoyxk/utils"
	"github.com/miekg/dns"
)

var defaultDNS = ""

var dnsCache = &cache{}

type cache struct {
	data map[string]*data
	mux  sync.RWMutex
}

type data struct {
	a []dns.RR
	t time.Time
}

func (c *cache) init() {
	c.data = make(map[string]*data)
}

func (c *cache) Set(domain string, a []dns.RR) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.data[domain] = &data{a: a, t: time.Now()}
}

func (c *cache) Get(domain string) []dns.RR {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if d, ok := c.data[domain]; ok {
		if time.Now().Sub(d.t) < time.Minute*3 {
			return d.a
		}
	}
	return nil
}

type handler struct{}

func (t *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name
		service, ok := app.DnsDomain[domain]
		if ok {

			if service.Status == config.Stop && service.Switch == nil {
				var ch, err = k8s.ForwardService(service)
				if err != nil {
					console.Error(err)
				} else {
					<-ch
				}
			}

			var ip = ""

			if service.Pod != nil {
				ip = service.Pod.IP
			}

			if service.Switch != nil {
				ip = service.Switch.Pod.IP
			}

			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1},
				A:   net.ParseIP(ip),
			})
		} else {
			var aCache = dnsCache.Get(domain)
			if aCache != nil {
				msg.Answer = aCache
			} else {
				var m = dns.Msg{}
				m.SetQuestion(domain, dns.TypeA)
				r, err := dns.Exchange(&m, defaultDNS+":53")
				if err != nil {
					tools.Exit(err)
				}

				var rr []dns.RR

				for _, i2 := range r.Answer {
					if i2.Header().Rrtype == dns.TypeA {
						var a = &dns.A{
							Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
							A:   net.ParseIP(i2.(*dns.A).A.String()),
						}
						rr = append(rr, a)
					}
				}

				msg.Answer = rr

				dnsCache.Set(domain, rr)
			}

		}
	}

	err := w.WriteMsg(&msg)
	if err != nil {
		console.Error(err)
	}
}

func AddNameServer() {
	GetDefaultNDS()

	if runtime.GOOS == "linux" {
		addNameServerLinux()
	} else if runtime.GOOS == "darwin" {
		addNameServerDarwin()
	} else if runtime.GOOS == "windows" {
		tools.Exit("not support windows")
	}
}

func GetDefaultNDS() {
	f, err := os.OpenFile("/etc/resolv.conf", os.O_RDWR, 0777)
	if err != nil {
		tools.Exit(err)
	}

	defer func() { _ = f.Close() }()

	var lines []string

	var reader = bufio.NewReader(f)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			tools.Exit(err)
		}

		lines = append(lines, line)
	}

	var si = -1

	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "nameserver") {
			si = i
			break
		}
	}

	if si != -1 {
		defaultDNS = strings.TrimSpace(strings.Split(lines[si], " ")[1])
	}

}

func addNameServerDarwin() {
	for domain := range app.DnsDomain {
		var model, _ = app.Temp.ReadFile("temp/resolv.conf")
		var res = tools.ReplaceString(
			string(model),
			[]string{"@domain", "@ip"},
			[]string{domain[:len(domain)-1], "127.0.0.1"},
		)

		err := utils.File.ReadFromString(res).WriteToPath(`/etc/resolver/` + domain + `local`)
		if err != nil {
			console.Error("dns: domain", domain, "create failed", err)
		} else {
			console.Info("dns: domain", domain, "create success")
		}
	}
}

func addNameServerLinux() {
	f, err := os.OpenFile("/etc/resolv.conf", os.O_RDWR, 0777)
	if err != nil {
		tools.Exit(err)
	}

	defer func() { _ = f.Close() }()

	var lines []string

	var reader = bufio.NewReader(f)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			tools.Exit(err)
		}

		lines = append(lines, line)
	}

	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "nameserver 127.0.0.1") {
			return
		}
	}

	var si = -1

	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "nameserver") {
			si = i
			break
		}
	}

	if si == -1 {
		si = 0
	}

	var newLines []string

	newLines = append(newLines, lines[:si]...)

	newLines = append(newLines, "nameserver 127.0.0.1\n")

	newLines = append(newLines, lines[si:]...)

	err = f.Truncate(0)
	if err != nil {
		tools.Exit(err)
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		tools.Exit(err)
	}

	var str = ""

	for i := 0; i < len(newLines); i++ {
		str += newLines[i]
	}

	_, err = f.WriteString(str)
	if err != nil {
		console.Error("dns: global create failed", err)
	} else {
		console.Info("dns: global create success")
	}
}

func DeleteNameServer() {
	if runtime.GOOS == "linux" {
		deleteNameServerLinux()
	} else if runtime.GOOS == "darwin" {
		deleteNameServerDarwin()
	} else if runtime.GOOS == "windows" {
		tools.Exit("not support windows")
	}
}

func deleteNameServerDarwin() {
	for domain := range app.DnsDomain {
		var err = os.RemoveAll(`/etc/resolver/` + domain + `local`)
		if err != nil {
			console.Error("dns: domain", domain, "delete failed", err)
		} else {
			console.Warning("dns: domain", domain, "delete success")
		}
	}
}

func deleteNameServerLinux() {
	f, err := os.OpenFile("/etc/resolv.conf", os.O_RDWR, 0777)
	if err != nil {
		tools.Exit(err)
	}

	defer func() { _ = f.Close() }()

	var lines []string

	var reader = bufio.NewReader(f)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			tools.Exit(err)
		}

		lines = append(lines, line)
	}

	var si = -1

	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "nameserver 127.0.0.1") {
			si = i
			break
		}
	}

	if si == -1 {
		return
	}

	err = f.Truncate(0)
	if err != nil {
		tools.Exit(err)
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		tools.Exit(err)
	}

	var str = ""

	for i := 0; i < len(lines); i++ {
		if i == si {
			continue
		}
		str += lines[i]
	}

	_, err = f.WriteString(str)
	if err != nil {
		console.Error("dns: global delete failed", err)
	} else {
		console.Warning("dns: global delete success")
	}
}

// StartDNS startDNS starts a DNS server on the given port.
func StartDNS(fn func()) {
	dnsCache.init()
	srv := &dns.Server{Addr: "127.0.0.1:" + strconv.Itoa(53), Net: "udp"}
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
