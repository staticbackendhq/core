package email

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

type AWSSES struct{}

func (AWSSES) Send(data SendMailData) error {
	if len(data.To) == 0 || strings.Index(data.To, "@") == -1 {
		return fmt.Errorf("empty To email")
	}

	if len(data.ReplyTo) == 0 {
		data.ReplyTo = data.From
	}

	charset := "UTF-8"

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Current.AWSRegion)},
	)
	if err != nil {
		return err
	}

	// Create an SES session.
	svc := ses.New(sess)

	from := fmt.Sprintf("%s <%s>", data.FromName, data.From)

	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(data.To),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(charset),
					Data:    aws.String(data.HTMLBody),
				},
				Text: &ses.Content{
					Charset: aws.String(charset),
					Data:    aws.String(data.TextBody),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(charset),
				Data:    aws.String(data.Subject),
			},
		},
		Source:           aws.String(from),
		ReplyToAddresses: aws.StringSlice([]string{data.ReplyTo}),
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}

	// Attempt to send the email.
	if _, err := svc.SendEmail(input); err != nil {
		return err
	}

	return nil
}
