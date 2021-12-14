package server

const (
	defaultServiceName = "alertEngine"
	defaultHttpPort    = 5050
)

const (
	SmtpHostEnv     = "SMTP_HOST"
	SmtpPortEnv     = "SMTP_PORT"
	SmtpAccountEnv  = "SMTP_ACCOUNT"
	SmtpPasswordEnv = "SMTP_PASSWORD"
)

const (
	BasePath      = "/api"
	AlertPath     = "/alert"
	SubscribePath = "/subscribe"
)
