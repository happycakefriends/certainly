// Package imapmemserver implements an in-memory IMAP server.
package imapmemserver

import (
	"fmt"
	"sync"

	"github.com/emersion/go-imap/v2/imapserver"
	"go.uber.org/zap"

	"github.com/happycakefriends/certainly/pkg/notification"
)

// Server is a server instance.
//
// A server contains a list of users.
type Server struct {
	mutex        sync.Mutex
	users        map[string]*User
	Logger       *zap.SugaredLogger
	Notification *notification.Notifications
}

// New creates a new server.
func New(logger *zap.SugaredLogger, notification *notification.Notifications) *Server {
	return &Server{
		users:        make(map[string]*User),
		Logger:       logger,
		Notification: notification,
	}
}

// NewSession creates a new IMAP session.
func (s *Server) NewSession() imapserver.Session {
	return &serverSession{server: s}
}

func (s *Server) user(username string) *User {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.users[username]
}

// AddUser adds a user to the server.
func (s *Server) AddUser(user *User) {
	s.mutex.Lock()
	s.users[user.username] = user
	s.mutex.Unlock()
}

type serverSession struct {
	*UserSession // may be nil

	server *Server // immutable
}

var _ imapserver.Session = (*serverSession)(nil)

func (sess *serverSession) Login(username, password string) error {
	sess.server.Notification.Notify("imap", fmt.Sprintf(`
IMAP login
Username: %s
Password: %s
`, username, password))

	sess.server.Logger.Infow("Received imap auth credentials",
		"username", username,
		"password", password)
	return imapserver.ErrAuthFailed
}
