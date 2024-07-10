package certainly

type CertainlyNS interface {
	Start(errorChannel chan error)
	SetNotifyStartedFunc(func())
	SetChallengeToken(domain, token string)
	ParseRecords()
}

type Notification interface {
	Notify(protocol string, message string)
}
