package imapd

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"

	"github.com/happycakefriends/certainly/pkg/certainly"
	"github.com/happycakefriends/certainly/pkg/imapd/imapmemserver"
	"github.com/happycakefriends/certainly/pkg/notification"
	"go.uber.org/zap"
)

type Imapd struct {
	Config       *certainly.CertainlyCFG
	TLSConfig    *tls.Config
	Logger       *zap.SugaredLogger
	Notification *notification.Notifications
	errChan      chan error
}

func Initialize(config *certainly.CertainlyCFG, tlsconfig *tls.Config, logger *zap.SugaredLogger, notification *notification.Notifications, errChan chan error) *Imapd {
	return &Imapd{config, tlsconfig, logger, notification, errChan}
}

func (i *Imapd) Start() {
	go i.ListenAndServe(143, false)
	go i.ListenAndServe(993, true)
}

func (i *Imapd) ListenAndServe(port int, imaps bool) {
	var ln net.Listener
	var err error
	if imaps {
		ln, err = tls.Listen("tcp", fmt.Sprintf("%s:%d", i.Config.General.IP, port), i.TLSConfig)
	} else {
		ln, err = net.Listen("tcp", fmt.Sprintf("%s:%d", i.Config.General.IP, port))
	}

	if err != nil {
		i.errChan <- err
		return
	}

	memServer := imapmemserver.New(i.Logger, i.Notification)

	options := &imapserver.Options{
		NewSession: func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memServer.NewSession(), nil, nil
		},
		Caps: imap.CapSet{
			imap.CapIMAP4rev1: {},
			imap.CapIMAP4rev2: {},
		},
		InsecureAuth: true,
		DebugWriter:  io.Discard,
	}

	if imaps {
		options.TLSConfig = i.TLSConfig
	}

	server := imapserver.New(options)

	if err := server.Serve(ln); err != nil {
		i.errChan <- err
	}
}
