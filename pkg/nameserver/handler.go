package nameserver

import (
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/happycakefriends/certainly/pkg/util"
	"github.com/miekg/dns"
)

func (n *Nameserver) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	// handle edns0
	opt := r.IsEdns0()
	if opt != nil {
		if opt.Version() != 0 {
			// Only EDNS0 is standardized
			m.MsgHdr.Rcode = dns.RcodeBadVers
			m.SetEdns0(512, false)
		} else {
			// We can safely do this as we know that we're not setting other OPT RRs within certainly.
			m.SetEdns0(512, false)
			if r.Opcode == dns.OpcodeQuery {
				n.readQuery(m, w.RemoteAddr().String())
			}
		}
	} else {
		if r.Opcode == dns.OpcodeQuery {
			n.readQuery(m, w.RemoteAddr().String())
		}
	}
	_ = w.WriteMsg(m)
}

func (n *Nameserver) readQuery(m *dns.Msg, remoteAddr string) {
	var authoritative = false
	for _, que := range m.Question {
		if rr, rc, auth, err := n.answer(que, remoteAddr); err == nil {
			if auth {
				authoritative = auth
			}
			m.MsgHdr.Rcode = rc
			m.Answer = append(m.Answer, rr...)
		}
	}
	m.MsgHdr.Authoritative = authoritative
	if authoritative {
		if m.MsgHdr.Rcode == dns.RcodeNameError {
			m.Ns = append(m.Ns, n.SOA)
		}
	}
}

func (n *Nameserver) answer(q dns.Question, remoteAddr string) ([]dns.RR, int, bool, error) {
	var rcode int
	var authoritative = n.isAuthoritative(q)
	rcode = dns.RcodeSuccess
	r, err := n.getRecord(q)

	if err != nil {
		rcode = dns.RcodeNameError
	}

	if q.Qtype == dns.TypeTXT {
		if n.isOwnChallenge(q.Name) {
			txtRRs, _ := n.answerOwnChallenge(q)
			r = append(r, txtRRs...)
		}
	}
	if q.Qtype == dns.TypeMX {
		if n.answeringForDomain(q.Name) {
			amx := new(dns.MX)
			amx.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeMX, Class: dns.ClassINET, Ttl: 1}
			amx.Mx = dns.Fqdn(n.Config.NS.DefaultDomain)
			amx.Preference = 10
			r = append(r, amx)
		}
	}
	if q.Qtype == dns.TypeA {
		if n.answeringForDomain(q.Name) {
			a := new(dns.A)
			a.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1}
			ipv4Addr, _, _ := net.ParseCIDR(n.Config.General.IP + "/32")
			a.A = ipv4Addr
			r = append(r, a)
		}
	}
	if q.Qtype == dns.TypeCNAME {
		if !util.HasApexDomain(q.Name, n.Config.NS.DefaultDomain) {
			// Do not answer CNAMES for the default domain to prevent endless loops because of some resolvers
			if n.answeringForDomain(q.Name) {
				cn := new(dns.CNAME)
				cn.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 1}
				cn.Target = fmt.Sprintf("%s.%s.", uuid.New().String(), n.Config.NS.DefaultDomain)
				r = append(r, cn)
				// Add the A answer for the CNAME target to the response
				a := new(dns.A)
				a.Hdr = dns.RR_Header{Name: cn.Target, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1}
				ipv4Addr, _, _ := net.ParseCIDR(n.Config.General.IP + "/32")
				a.A = ipv4Addr
				r = append(r, a)
			}
		}
	}
	if len(r) > 0 {
		// Make sure that we return NOERROR if there were dynamic records for the domain
		rcode = dns.RcodeSuccess
	}
	n.Notification.Notify("dns", fmt.Sprintf(`
DNS question from: %s
Type:   %s
Rcode:  %s
Domain: %s`,
		remoteAddr, dns.TypeToString[q.Qtype], dns.RcodeToString[rcode], q.Name))

	n.Logger.Infow("Answering question for domain",
		"qtype", dns.TypeToString[q.Qtype],
		"domain", q.Name,
		"rcode", dns.RcodeToString[rcode],
		"remoteAddr", remoteAddr)
	return r, rcode, authoritative, nil
}

func (n *Nameserver) isAuthoritative(q dns.Question) bool {
	if n.answeringForDomain(q.Name) {
		return true
	}
	domainParts := strings.Split(strings.ToLower(q.Name), ".")
	for i := range domainParts {
		if n.answeringForDomain(strings.Join(domainParts[i:], ".")) {
			return true
		}
	}
	return false
}

// isOwnChallenge checks if the query is for the domain of this certainly instance. Used for answering its own ACME challenges
func (n *Nameserver) isOwnChallenge(name string) bool {
	domainParts := strings.SplitN(name, ".", 2)
	if len(domainParts) == 2 {
		if strings.ToLower(domainParts[0]) == "_acme-challenge" {
			domain := strings.ToLower(domainParts[1])
			if !strings.HasSuffix(domain, ".") {
				domain = domain + "."
			}
			if n.inOwnDomains(domain) {
				return true
			}
		}
	}
	return false
}

func (n *Nameserver) inOwnDomains(name string) bool {
	for _, d := range n.OwnDomains {
		if strings.HasSuffix(strings.ToLower(name), d) {
			return true
		}
	}
	return false
}

// answeringForDomain checks if we have any records for a domain
func (n *Nameserver) answeringForDomain(name string) bool {
	if n.inOwnDomains(name) {
		return true
	}
	_, ok := n.Domains[strings.ToLower(name)]
	return ok
}

func (n *Nameserver) getRecord(q dns.Question) ([]dns.RR, error) {
	var rr []dns.RR
	var cnames []dns.RR
	domain, ok := n.Domains[strings.ToLower(q.Name)]
	if !ok {
		return rr, fmt.Errorf("no records for domain %s", q.Name)
	}
	for _, ri := range domain.Records {
		if ri.Header().Rrtype == q.Qtype {
			rr = append(rr, ri)
		}
		if ri.Header().Rrtype == dns.TypeCNAME {
			cnames = append(cnames, ri)
		}
	}
	if len(rr) == 0 {
		return cnames, nil
	}
	return rr, nil
}
