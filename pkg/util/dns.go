package util

import (
	"fmt"

	"github.com/happycakefriends/certainly/pkg/certainly"
	"github.com/miekg/dns"
)

// GetA fetches the A records for a domain
func GetA(domain string) (certainly.UpstreamNSRecord, error) {
	record := certainly.UpstreamNSRecord{IPv6: false}
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	in, err := dns.Exchange(msg, "8.8.8.8:53")
	if err != nil {
		return record, err
	}

	for _, ans := range in.Answer {
		if a, ok := ans.(*dns.A); ok {
			record.Addr = a.A.String()
			return record, nil
		} else {
			return record, fmt.Errorf("unexpected record returned with A query to domain %s", domain)
		}
	}
	return record, fmt.Errorf("no A record found")
}

// GetA fetches the AAAA record for a domain
func GetAAAA(domain string) (certainly.UpstreamNSRecord, error) {
	record := certainly.UpstreamNSRecord{IPv6: true}
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeAAAA)
	msg.RecursionDesired = true

	in, err := dns.Exchange(msg, "8.8.8.8:53")
	if err != nil {
		return record, err
	}

	for _, ans := range in.Answer {
		if a, ok := ans.(*dns.AAAA); ok {
			record.Addr = a.AAAA.String()
			return record, nil
		} else {
			return record, fmt.Errorf("unexpected record returned with AAAA query to domain %s", domain)
		}
	}
	return record, fmt.Errorf("no AAAA record found")
}

// GetCNAME fetches the CNAME record for a domain
func GetCNAME(domain string) (certainly.UpstreamNSRecord, error) {
	record := certainly.UpstreamNSRecord{}
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeCNAME)
	msg.RecursionDesired = true

	in, err := dns.Exchange(msg, "8.8.8.8:53")
	if err != nil {
		return record, err
	}

	for _, ans := range in.Answer {
		if a, ok := ans.(*dns.CNAME); ok {
			record.Addr = a.Target
			return record, nil
		} else {
			return record, fmt.Errorf("unexpected record returned with CNAME query to domain %s", domain)
		}
	}
	return record, fmt.Errorf("no CNAME record found")
}

// ExistsUpstream checks if there is a valid A, AAAA or CNAME record in the upstream DNS zone
func ExistsUpstream(domain string) bool {
	_, err := GetA(domain)
	if err == nil {
		return true
	}
	_, err = GetAAAA(domain)
	if err == nil {
		return true
	}
	_, err = GetCNAME(domain)
	return err == nil
}
