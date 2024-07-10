package notification

import (
	"regexp"

	"github.com/happycakefriends/certainly/pkg/certainly"
	"go.uber.org/zap"
)

type Notifications struct {
	Engines []certainly.Notification
	Config  *certainly.CertainlyCFG
	Logger  *zap.SugaredLogger
}

func Initialize(config *certainly.CertainlyCFG, logger *zap.SugaredLogger) *Notifications {
	notifications := &Notifications{Config: config, Logger: logger}
	notifications.Engines = make([]certainly.Notification, 0)
	if config.Notification.Slack {
		slack, err := NewSlack(config, logger)
		if err != nil {
			logger.Errorf("Failed to initialize Slack notification engine",
				"error", err)
		} else {
			notifications.Engines = append(notifications.Engines, slack)
		}
	}
	return notifications
}

func (n *Notifications) Notify(protocol string, message string) {
	for _, engine := range n.Engines {
		engine.Notify(protocol, message)
	}
}

func matchAnyFilter(data string, filters []string) bool {
	for _, filter := range filters {
		if filter == "" {
			continue
		}
		if match, _ := regexp.MatchString(filter, data); match {
			return true
		}
	}
	return false
}
