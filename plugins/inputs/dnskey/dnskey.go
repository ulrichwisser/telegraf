package dnskey

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"regexp"
	"time"

	"github.com/miekg/dns"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type Dnskey struct {
	// Domains or subdomains to query
	Domains []string

	// Resolvers
	Resolvers []string

	// Dns query timeout in seconds. 0 means no timeout
	Timeout int
}

type dnsresponse struct {
	queryTime float64
	msg       dns.Msg
}

var sampleConfig = `
  ## Domains to query.
  # domains = ["ietf.org", "icann.org"]

  ## Resolvers (to specify port write ipv4:53 or [ipv6]:53)
	# resolvers = ["8.8.8.8", "8.8.4.4"]

  ## Query timeout in seconds.
  # timeout = 2
`

func (d *Dnskey) SampleConfig() string {
	return sampleConfig
}

func (d *Dnskey) Description() string {
	return "Query (through system resolver) for DNSKEYs for a given domain"
}

func (d *Dnskey) Gather(acc telegraf.Accumulator) error {
	err := d.setDefaultValues()
	if err != nil {
		acc.AddError(err)
		return nil
	}

	for _, domain := range d.Domains {
		server := d.getResolver()
		dnsMsg, dnsQueryTime, err := d.getDnskey(domain, server)
		if err != nil {
			acc.AddError(err)
			return nil
		}
		if dnsMsg.Rcode != dns.RcodeSuccess {
			acc.AddError(fmt.Errorf("Query failed! Rcode %d  Querying: %s for %s", dnsMsg.Rcode, server, domain))
			return nil
		}

		for _, rr := range dnsMsg.Answer {
			if rr.Header().Rrtype != dns.TypeDNSKEY {
				continue
			}

			tags := map[string]string{
				"domain":    domain,
				"server":    server,
				"keytag":    fmt.Sprintf("%d", rr.(*dns.DNSKEY).KeyTag()),
				"algorithm": algorithmName(rr.(*dns.DNSKEY).Algorithm),
				"key_type":  keyType(rr.(*dns.DNSKEY).Flags),
			}

			fields := map[string]interface{}{
				"query_time_ms": dnsQueryTime,
			}
			acc.AddFields("dnskey", fields, tags)
		}
	}

	return nil
}

func (d *Dnskey) getResolver() string {
	if d.Resolvers == nil || len(d.Resolvers) == 0 {
		return ""
	}
	return d.Resolvers[rand.Intn(len(d.Resolvers))]
}

func (d *Dnskey) setDefaultValues() error {

	if len(d.Domains) == 0 {
		d.Domains = []string{"."}
	}

	// get system resolvers
	if d.Resolvers == nil || len(d.Resolvers) == 0 {
		clientconfig, err := dns.ClientConfigFromFile(`/etc/resolv.conf`)
		if err != nil {
			return errors.New(`Could not read /etc/resolv.conf`)
		}
		d.Resolvers = clientconfig.Servers
	}

	// reformat resolvers in name:port format
	m := regexp.MustCompile(`^(\d+(\.\d+){3}:\d+)|(\[.*\]:\d+)$`)
	for i := range d.Resolvers {
		if !m.MatchString(d.Resolvers[i]) {
			d.Resolvers[i] = net.JoinHostPort(d.Resolvers[i], "53")
		}
	}
	// default timeout 2 seconds
	if d.Timeout == 0 {
		d.Timeout = 2
	}

	return nil
}

func algorithmName(alg uint8) string {
	var str string
	var ok bool
	if str, ok = dns.AlgorithmToString[alg]; !ok {
		str = fmt.Sprintf("%d", alg)
	}
	return str
}

func keyType(flags uint16) string {
	if flags == 256 {
		return "ZSK"
	}
	if flags == 257 {
		return "KSK"
	}
	return fmt.Sprintf("%d", flags)
}

func (d *Dnskey) getDnskey(domain string, server string) (*dns.Msg, time.Duration, error) {

	c := new(dns.Client)
	c.ReadTimeout = time.Duration(d.Timeout) * time.Second

	m := new(dns.Msg)
	m.SetEdns0(4096, true)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeDNSKEY)

	return c.Exchange(m, server)
}

func init() {
	inputs.Add("dnskey", func() telegraf.Input {
		return &Dnskey{}
	})
}
