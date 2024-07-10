package smtpd

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"

	"github.com/happycakefriends/certainly/pkg/certainly"
	"github.com/happycakefriends/certainly/pkg/notification"
	"github.com/mhale/smtpd"
	"go.uber.org/zap"
)

type Smtpd struct {
	Config       *certainly.CertainlyCFG
	TLSConfig    *tls.Config
	Logger       *zap.SugaredLogger
	Notification *notification.Notifications
	errChan      chan error
}

func Initialize(config *certainly.CertainlyCFG, tlsconfig *tls.Config, logger *zap.SugaredLogger, notification *notification.Notifications, errChan chan error) *Smtpd {
	return &Smtpd{config, tlsconfig, logger, notification, errChan}
}

func (s *Smtpd) Start() {
	// SMTP
	go s.ListenAndServe(25, false)
	// Submission
	go s.ListenAndServe(587, false)
	// SMTPS
	go s.ListenAndServe(465, true)
}

func (s *Smtpd) mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		s.Logger.Errorw("Error reading mail message",
			"error", err,
			"remoteAddr", origin.String())
		return err
	}
	subject := msg.Header.Get("Subject")
	s.Logger.Infow("Received mail",
		"remoteAddr", origin.String(),
		"from", from,
		"to", to[0],
		"subject", subject,
		"data", string(data))
	return nil
}

func (s *Smtpd) rcptHandler(remoteAddr net.Addr, from string, to string) bool {
	// Well certainly sir, we will accept all mail
	return true
}

func (s *Smtpd) authHandler(remoteAddr net.Addr, mechanism string, username []byte, password []byte, shared []byte) (bool, error) {
	// Oh, absolutely, this is a valid user. Let me log the information for you
	s.Notification.Notify("smtp", fmt.Sprintf(`
SMTP credentials from: %s
Auth Mechanism: %s
Username: %s
Password: %s
Shared secret (if any): %s`,
		remoteAddr.String(), mechanism,
		string(username), string(password), string(shared)))
	s.Logger.Infow("Received smtp auth credentials",
		"remoteAddr", remoteAddr.String(),
		"mechanism", mechanism,
		"username", string(username),
		"password", string(password),
		"shared", string(shared))
	return true, nil
}

func (s *Smtpd) ListenAndServe(port int, smtps bool) {
	srv := &smtpd.Server{
		AuthMechs:   map[string]bool{"PLAIN": true, "LOGIN": true},
		Addr:        fmt.Sprintf("%s:%d", s.Config.General.IP, port),
		Handler:     s.mailHandler,
		HandlerRcpt: s.rcptHandler,
		Appname:     "HCF Certainly SMTPD v0.1",
		AuthHandler: s.authHandler,
		TLSListener: smtps,
		Hostname:    "",
	}
	s.errChan <- srv.ListenAndServe()
}
