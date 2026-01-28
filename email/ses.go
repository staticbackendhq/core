package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type AWSSES struct{}

func (AWSSES) Send(data SendMailData) error {
	if len(data.To) == 0 || !strings.Contains(data.To, "@") {
		return fmt.Errorf("empty To email")
	}

	if len(data.ReplyTo) == 0 {
		data.ReplyTo = data.From
	}

	charset := "UTF-8"

	region := strings.TrimSpace(config.Current.S3Region)
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return err
	}

	// Create an SES client.
	svc := ses.NewFromConfig(cfg)

	from := fmt.Sprintf("%s <%s>", data.FromName, data.From)

	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &types.Destination{
			CcAddresses: []string{},
			ToAddresses: []string{
				data.To,
			},
		},
		Message: &types.Message{
			Body: &types.Body{
				Html: &types.Content{
					Charset: aws.String(charset),
					Data:    aws.String(data.HTMLBody),
				},
				Text: &types.Content{
					Charset: aws.String(charset),
					Data:    aws.String(data.TextBody),
				},
			},
			Subject: &types.Content{
				Charset: aws.String(charset),
				Data:    aws.String(data.Subject),
			},
		},
		Source:           aws.String(from),
		ReplyToAddresses: []string{data.ReplyTo},
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}

	// Attempt to send the email.
	if _, err := svc.SendEmail(context.TODO(), input); err != nil {
		return err
	}

	return nil
}
