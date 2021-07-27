package internal

// SendMailData contains necessary fields to send an email
type SendMailData struct {
	From     string
	FromName string
	To       string
	ToName   string
	Subject  string
	HTMLBody string
	TextBody string
	ReplyTo  string
}

// Mailer is used to have different implementation for sending email
type Mailer interface {
	Send(SendMailData) error
}
