package main

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

// sendmail uses Amazon SES to send an HTML email, it will convert body to text automatically
func sendMail(toEmail, toName, fromEmail, fromName, subject, body string, files []string) error {
	if len(toEmail) == 0 || strings.Index(toEmail, "@") == -1 {
		return fmt.Errorf("empty to email")
	}

	charset := "UTF-8"

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		return err
	}

	// Create an SES session.
	svc := ses.New(sess)

	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(toEmail),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(charset),
					Data:    aws.String(body),
				},
				Text: &ses.Content{
					Charset: aws.String(charset),
					Data:    aws.String(stripHTML(body)),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(charset),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(fromEmail),
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}

	// Attempt to send the email.
	if _, err := svc.SendEmail(input); err != nil {
		return err
	}

	return nil
}

// stripHTML returns a version of a string with no HTML tags.
func stripHTML(s string) string {
	output := ""

	// if we have a full html page we only need the body
	startBody := strings.Index(s, "<body")
	if startBody > -1 {
		endBody := strings.Index(s, "</body>")
		// try to find the end of the <body tag
		for i := startBody; i < endBody; i++ {
			if s[i] == '>' {
				startBody = i
				break
			}
		}

		if startBody < endBody {
			s = s[startBody:endBody]
		}
	}

	// Shortcut strings with no tags in them
	if !strings.ContainsAny(s, "<>") {
		output = s
	} else {
		// Removing line feeds
		s = strings.Replace(s, "\n", "", -1)

		// Then replace line breaks with newlines, to preserve that formatting
		s = strings.Replace(s, "</h1>", "\n\n", -1)
		s = strings.Replace(s, "</h2>", "\n\n", -1)
		s = strings.Replace(s, "</h3>", "\n\n", -1)
		s = strings.Replace(s, "</h4>", "\n\n", -1)
		s = strings.Replace(s, "</h5>", "\n\n", -1)
		s = strings.Replace(s, "</h6>", "\n\n", -1)
		s = strings.Replace(s, "</p>", "\n", -1)
		s = strings.Replace(s, "<br>", "\n", -1)
		s = strings.Replace(s, "<br/>", "\n", -1)
		s = strings.Replace(s, "<br />", "\n", -1)

		// Walk through the string removing all tags
		b := bytes.NewBufferString("")
		inTag := false
		for _, r := range s {
			switch r {
			case '<':
				inTag = true
			case '>':
				inTag = false
			default:
				if !inTag {
					b.WriteRune(r)
				}
			}
		}
		output = b.String()
	}

	// Remove a few common harmless entities, to arrive at something more like plain text
	output = strings.Replace(output, "&#8216;", "'", -1)
	output = strings.Replace(output, "&#8217;", "'", -1)
	output = strings.Replace(output, "&#8220;", "\"", -1)
	output = strings.Replace(output, "&#8221;", "\"", -1)
	output = strings.Replace(output, "&nbsp;", " ", -1)
	output = strings.Replace(output, "&quot;", "\"", -1)
	output = strings.Replace(output, "&apos;", "'", -1)

	// Translate some entities into their plain text equivalent (for example accents, if encoded as entities)
	output = html.UnescapeString(output)

	// In case we have missed any tags above, escape the text - removes <, >, &, ' and ".
	output = template.HTMLEscapeString(output)

	// After processing, remove some harmless entities &, ' and " which are encoded by HTMLEscapeString
	output = strings.Replace(output, "&#34;", "\"", -1)
	output = strings.Replace(output, "&#39;", "'", -1)
	output = strings.Replace(output, "&amp; ", "& ", -1)     // NB space after
	output = strings.Replace(output, "&amp;amp; ", "& ", -1) // NB space after

	return output
}
