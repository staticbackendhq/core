package main

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"html/template"
	"strings"

	ses "github.com/sourcegraph/go-ses"
	"gopkg.in/gomail.v2"
)

// SendMail uses Amazon SES to send an HTML email, it will convert body to text automatically
func sendMail(toEmail, toName, fromEmail, fromName, subject, body, replyTo string) error {
	if len(toEmail) == 0 || strings.Index(toEmail, "@") == -1 {
		return fmt.Errorf("empty to email")
	}

	m := gomail.NewMessage()
	m.SetHeader("To", toEmail)
	if len(replyTo) > 0 {
		m.SetHeader("From", fromName+" <"+replyTo+">")
		m.SetHeader("Reply-To", replyTo)
	} else {
		m.SetHeader("From", fromName+" <"+fromEmail+">")
	}
	m.SetHeader("Subject", subject)
	m.AddAlternative("text/plain", stripHTML(body))
	m.SetBody("text/html", body)

	var b bytes.Buffer
	m.WriteTo(&b)

	res, err := ses.EnvConfig.SendRawEmail(b.Bytes())
	if err != nil {
		return err
	} else if len(res) == 0 {
		return errors.New("No email id returned by Amazon SES")
	}

	return nil

	/*res, err := ses.EnvConfig.SendEmailHTML(fromEmail, toEmail, subject, StripHTML(body), body)
	if err != nil {
		return err
	} else if len(res) == 0 {
		return errors.New("No email id returned by Amazon SES")
	}
	return nil*/
}

// StripHTML returns a version of a string with no HTML
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
