package email

const (
	MailProviderDev = "dev"
	MailProviderSES = "ses"
)

// SendMailData contains necessary fields to send an email
type SendMailData struct {
	From     string `json:"from"`
	FromName string `json:"fromName"`
	To       string `json:"to"`
	ToName   string `json:"toName"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"htmlBody"`
	TextBody string `json:"textBody"`
	ReplyTo  string `json:"replyTo"`

	Body string `json:"body"`
}

// Mailer is used to have different implementation for sending email
type Mailer interface {
	Send(SendMailData) error
}
