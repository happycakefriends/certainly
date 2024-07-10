package nameserver

import (
	"strings"

	"github.com/miekg/dns"
)

// SetOwnAuthKey sets the ACME challenge token for completing dns validation for certainly server itself
func (n *Nameserver) SetChallengeToken(domain, key string) {
	if !strings.HasSuffix(domain, ".") {
		domain = domain + "."
	}
	n.ownChallenges[domain] = key
}

// answerOwnChallenge answers to ACME challenge for certainly own certificate
func (n *Nameserver) answerOwnChallenge(q dns.Question) ([]dns.RR, error) {
	r := new(dns.TXT)
	r.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 1}
	r.Txt = append(r.Txt, n.getChallengeToken(q.Name))
	return []dns.RR{r}, nil
}

func (n *Nameserver) getChallengeToken(domain string) string {
	val, ok := n.ownChallenges[strings.ToLower(domain)]
	if ok {
		return val
	}
	return ""
}
