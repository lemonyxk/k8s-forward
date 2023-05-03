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

	"github.com/lemonyxk/console"
	"github.com/lemonyxk/k8s-forward/app"
	"github.com/lemonyxk/k8s-forward/config"
	"github.com/lemonyxk/k8s-forward/k8s"
	"github.com/lemonyxk/k8s-forward/tools"
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
	ip string
	t  time.Time
}

func (c *cache) init() {
	c.data = make(map[string]*data)
}

func (c *cache) Set(domain string, ip string, a []dns.RR) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.data[domain] = &data{a: a, ip: ip, t: time.Now()}
}

func (c *cache) Get(domain string) (string, []dns.RR) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if d, ok := c.data[domain]; ok {
		if time.Now().Sub(d.t) < time.Minute*3 {
			return d.ip, d.a
		}
	}
	return "", nil
}

type handler struct{}

func (t *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	var msg = &dns.Msg{}
	msg.SetReply(r)

	var domain = msg.Question[0].Name
	var tp = r.Question[0].Qtype
	// && tp != dns.TypeHTTPS
	if tp != dns.TypeA {
		err := w.WriteMsg(msg)
		if err != nil {
			console.Error(err)
		}
		return
	}

	msg.Authoritative = true

	// cache
	var ip, aCache = dnsCache.Get(domain)
	if aCache != nil {
		msg.Answer = aCache
		doHandler(w, domain, ip, msg)
		return
	}

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

		var rr []dns.RR
		var a = &dns.A{
			Hdr: dns.RR_Header{Name: domain, Rrtype: tp, Class: dns.ClassINET, Ttl: 1},
			A:   net.ParseIP(ip),
		}
		rr = append(rr, a)
		msg.Answer = rr
		dnsCache.Set(domain, ip, rr)

		doHandler(w, domain, ip, msg)
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

	if res == nil {
		doHandler(w, domain, ip, msg)
		return
	}

	var rr []dns.RR
	for _, i2 := range res.Answer {
		if i2.Header().Rrtype == tp {
			ip = i2.(*dns.A).A.String()
			var a = &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: tp, Class: dns.ClassINET, Ttl: 60},
				A:   net.ParseIP(ip),
			}
			rr = append(rr, a)
		}
	}

	msg.Answer = rr
	dnsCache.Set(domain, ip, rr)

	doHandler(w, domain, ip, msg)
	return
}

func doHandler(w dns.ResponseWriter, domain, ip string, msg *dns.Msg) {
	console.Info(domain[0:len(domain)-1], "from:", w.RemoteAddr().String(), "to:", ip)
	err := w.WriteMsg(msg)
	if err != nil {
		console.Error(err)
	}
}

func AddNameServer() {
	switch runtime.GOOS {
	case "linux":
		GetDefaultNDS()
		addNameServerLinux()
	case "darwin":
		GetDefaultNDS()
		addNameServerDarwin()
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
		console.Exit(err)
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
			console.Exit(err)
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
		console.Exit(err)
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		console.Exit(err)
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
	switch runtime.GOOS {
	case "linux":
		deleteNameServerLinux()
	case "darwin":
		deleteNameServerDarwin()
	default:
		console.Exit("not support windows")
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
		console.Exit(err)
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
			console.Exit(err)
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
		console.Exit(err)
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		console.Exit(err)
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

// StartDNS starts a DNS server on the given port.
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
