package notification

import (
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"

	"github.com/happycakefriends/certainly/pkg/certainly"
)

var (
	MSG_DATA_TEMPLATE = "```%s```\n------------------------------------------------------------"
)

type Slack struct {
	SlackToken          string
	SlackDefaultChannel string
	SlackHTTP           bool
	SlackHTTPChannel    string
	SlackHTTPFilters    []string
	SlackDNS            bool
	SlackDNSChannel     string
	SlackDNSFilters     []string
	SlackSMTP           bool
	SlackSMTPChannel    string
	SlackSMTPFilters    []string
	SlackIMAP           bool
	SlackIMAPChannel    string
	SlackIMAPFilters    []string
	Client              *slack.Client
	Logger              *zap.SugaredLogger
}

func NewSlack(config *certainly.CertainlyCFG, logger *zap.SugaredLogger) (*Slack, error) {
	s := &Slack{Logger: logger}
	if len(config.Notification.SlackToken) > 1 {
		s.SlackToken = config.Notification.SlackToken
	} else {
		return &Slack{}, fmt.Errorf("slack token not set")
	}
	if len(config.Notification.SlackDefaultChannel) > 1 {
		s.SlackDefaultChannel = config.Notification.SlackDefaultChannel
	} else {
		return &Slack{}, fmt.Errorf("slack default channel not set")
	}
	if len(config.Notification.SlackHTTPChannel) > 1 {
		s.SlackHTTPChannel = config.Notification.SlackHTTPChannel
	} else {
		s.SlackHTTPChannel = s.SlackDefaultChannel
	}
	if len(config.Notification.SlackDNSChannel) > 1 {
		s.SlackDNSChannel = config.Notification.SlackDNSChannel
	} else {
		s.SlackDNSChannel = s.SlackDefaultChannel
	}
	if len(config.Notification.SlackSMTPChannel) > 1 {
		s.SlackSMTPChannel = config.Notification.SlackSMTPChannel
	} else {
		s.SlackSMTPChannel = s.SlackDefaultChannel
	}
	if len(config.Notification.SlackIMAPChannel) > 1 {
		s.SlackIMAPChannel = config.Notification.SlackIMAPChannel
	} else {
		s.SlackIMAPChannel = s.SlackDefaultChannel
	}
	s.SlackDNSFilters, s.SlackHTTPFilters, s.SlackSMTPFilters, s.SlackIMAPFilters = config.Notification.DNSFilters, config.Notification.HTTPFilters, config.Notification.SMTPFilters, config.Notification.IMAPFilters
	s.SlackHTTP = config.Notification.HTTP
	s.SlackDNS = config.Notification.DNS
	s.SlackSMTP = config.Notification.SMTP
	s.SlackIMAP = config.Notification.IMAP
	s.Client = slack.New(s.SlackToken)
	// Test slack auth and connection
	_, err := s.Client.AuthTest()
	if err != nil {
		return s, fmt.Errorf("failed to authenticate with Slack: %s", err)
	}
	s.Notify("default", "Slack notification engine initialized")
	return s, nil
}

func (s *Slack) Notify(protocol string, message string) {
	channel := s.SlackDefaultChannel
	if protocol == "dns" {
		if !s.SlackDNS {
			return
		}
		if matchAnyFilter(message, s.SlackDNSFilters) {
			return
		}
		channel = s.SlackDNSChannel
	}
	if protocol == "http" {
		if !s.SlackHTTP {
			return
		}
		if matchAnyFilter(message, s.SlackHTTPFilters) {
			return
		}
		channel = s.SlackHTTPChannel
	}
	if protocol == "smtp" {
		if !s.SlackSMTP {
			return
		}
		if matchAnyFilter(message, s.SlackSMTPFilters) {
			return
		}
		channel = s.SlackSMTPChannel
	}
	if protocol == "imap" {
		if !s.SlackIMAP {
			return
		}
		if matchAnyFilter(message, s.SlackIMAPFilters) {
			return
		}
		channel = s.SlackIMAPChannel
	}
	_, _, err := s.Client.PostMessage(channel, slack.MsgOptionText(formatSlackMessage(message), true))
	if err != nil {
		s.Logger.Errorw("Failed to send message to Slack",
			"error", err)
	}
}

func formatSlackMessage(message string) string {
	message = strings.ReplaceAll(message, "```", "` ` `")
	t := time.Now()
	return t.Format(time.RFC3339) + "\n" + fmt.Sprintf(MSG_DATA_TEMPLATE, message)
}
