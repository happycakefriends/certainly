package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"regexp"
	"strings"

	"github.com/caddyserver/certmagic"
	"github.com/happycakefriends/certainly/pkg/certainly"
	"github.com/happycakefriends/certainly/pkg/util"
	"go.uber.org/zap"
)

func setupTLS(dnsservers []certainly.CertainlyNS, config *certainly.CertainlyCFG, sugar *zap.SugaredLogger) (*tls.Config, error) {
	provider := certainly.NewChallengeProvider(dnsservers)
	certmagic.Default.Logger = sugar.Desugar()
	storage := certmagic.FileStorage{Path: config.General.ACMECacheDir}

	// Set up certmagic for getting certificates via dns-01 challenge
	certmagic.DefaultACME.DNS01Solver = &provider
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Logger = sugar.Desugar()
	certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
	certmagic.DefaultACME.DisableHTTPChallenge = true
	certmagic.DefaultACME.Email = ""
	certmagic.Default.OnDemand = new(certmagic.OnDemandConfig)
	certmagic.Default.OnDemand.DecisionFunc = func(ctx context.Context, name string) error {
		if ShouldFilterTLS(config, name) {
			return fmt.Errorf("not allowed due to tls filter configuration")
		}
		for _, domain := range config.NS.Domains {
			if strings.HasSuffix(name, fmt.Sprintf(".%s", domain)) || name == domain {
				if util.ShouldRewrite(name, config.Rewrites) {
					if config.General.TLSUpstreamCheck && !util.ExistsUpstream(util.ReplaceApex(name, config.Rewrites)) {
						return fmt.Errorf("no valid upstream record found for domain %s", name)
					}
				}
				return nil
			}
		}
		return fmt.Errorf("not allowed")
	}

	magicConf := certmagic.Default
	magicConf.Logger = sugar.Desugar()
	magicConf.Storage = &storage
	magicConf.DefaultServerName = config.NS.DefaultDomain
	// Make sure we're requesting wildcard certificates for all subdomains
	magicConf.SubjectTransformer = func(ctx context.Context, name string) string {
		if certainly.IsManagedApex(name, config.NS.Domains) {
			return name
		}
		return certainly.TransformToWildcard(name)
	}
	magicCache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(cert certmagic.Certificate) (*certmagic.Config, error) {
			return &magicConf, nil
		},
		Logger: sugar.Desugar(),
	})

	magic := certmagic.New(magicCache, magicConf)

	// TLS config setup
	magictls := magic.TLSConfig()
	magictls.MinVersion = tls.VersionTLS12
	magictls.NextProtos = append([]string{"http/1.1", "h2", "http/1.0"}, magictls.NextProtos...)
	magictls.GetCertificate = magic.GetCertificate

	err := magic.ManageSync(context.Background(), certainly.WildcardDomains(config.NS.Domains))
	return magictls, err
}

func ShouldFilterTLS(config *certainly.CertainlyCFG, domain string) bool {
	for _, filter := range config.General.TLSFilters {
		if match, _ := regexp.MatchString(filter, domain); match {
			return true
		}
	}
	return false
}
