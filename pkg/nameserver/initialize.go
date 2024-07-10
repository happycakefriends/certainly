package nameserver

import (
	"fmt"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"go.uber.org/zap"

	"github.com/happycakefriends/certainly/pkg/certainly"
	"github.com/happycakefriends/certainly/pkg/notification"
)

// Records is a slice of ResourceRecords
type Records struct {
	Records []dns.RR
}

type Nameserver struct {
	Config            *certainly.CertainlyCFG
	Logger            *zap.SugaredLogger
	Notification      *notification.Notifications
	Server            *dns.Server
	OwnDomains        []string
	NotifyStartedFunc func()
	SOA               dns.RR
	ownChallenges     map[string]string
	Domains           map[string]Records
	errChan           chan error
}

func InitAndStart(config *certainly.CertainlyCFG, logger *zap.SugaredLogger, notification *notification.Notifications, errChan chan error) []certainly.CertainlyNS {
	dnsservers := make([]certainly.CertainlyNS, 0)
	waitLock := sync.Mutex{}
	if strings.HasPrefix(config.NS.Proto, "both") {
		// Handle the case where DNS server should be started for both udp and tcp
		udpProto := "udp"
		tcpProto := "tcp"
		if strings.HasSuffix(config.NS.Proto, "4") {
			udpProto += "4"
			tcpProto += "4"
		} else if strings.HasSuffix(config.NS.Proto, "6") {
			udpProto += "6"
			tcpProto += "6"
		}
		dnsServerUDP := NewDNSServer(config, logger, udpProto, notification)
		dnsservers = append(dnsservers, dnsServerUDP)
		dnsServerUDP.ParseRecords()
		dnsServerTCP := NewDNSServer(config, logger, tcpProto, notification)
		dnsservers = append(dnsservers, dnsServerTCP)
		dnsServerTCP.ParseRecords()
		// wait for the server to get started to proceed
		waitLock.Lock()
		dnsServerUDP.SetNotifyStartedFunc(waitLock.Unlock)
		go dnsServerUDP.Start(errChan)
		waitLock.Lock()
		dnsServerTCP.SetNotifyStartedFunc(waitLock.Unlock)
		go dnsServerTCP.Start(errChan)
		waitLock.Lock()
	} else {
		dnsServer := NewDNSServer(config, logger, config.NS.Proto, notification)
		dnsservers = append(dnsservers, dnsServer)
		dnsServer.ParseRecords()
		waitLock.Lock()
		dnsServer.SetNotifyStartedFunc(waitLock.Unlock)
		go dnsServer.Start(errChan)
		waitLock.Lock()
	}
	return dnsservers
}

// NewDNSServer parses the DNS records from config and returns a new DNSServer struct
func NewDNSServer(config *certainly.CertainlyCFG, logger *zap.SugaredLogger, proto string, notifications *notification.Notifications) certainly.CertainlyNS {
	//		dnsServerTCP := NewDNSServer(DB, Config.General.Listen, tcpProto, Config.General.Domain)
	server := Nameserver{Config: config, Logger: logger, Notification: notifications}
	server.Server = &dns.Server{Addr: fmt.Sprintf("%s:%s", config.General.IP, config.NS.Port), Net: proto}
	od := []string{}
	for _, d := range config.NS.Domains {
		if !strings.HasSuffix(d, ".") {
			d = d + "."
		}
		od = append(od, strings.ToLower(d))
	}
	server.OwnDomains = od
	server.ownChallenges = make(map[string]string)
	server.Domains = make(map[string]Records)
	return &server
}

func (n *Nameserver) Start(errorChannel chan error) {
	n.errChan = errorChannel
	dns.HandleFunc(".", n.handleRequest)
	n.Logger.Infow("Starting DNS listener",
		"addr", n.Server.Addr,
		"proto", n.Server.Net)
	if n.NotifyStartedFunc != nil {
		n.Server.NotifyStartedFunc = n.NotifyStartedFunc
	}
	err := n.Server.ListenAndServe()
	if err != nil {
		errorChannel <- err
	}
}

func (n *Nameserver) SetNotifyStartedFunc(fun func()) {
	n.Server.NotifyStartedFunc = fun
}
