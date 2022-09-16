package email

import (
	"fmt"
)

type Dev struct{}

func (d Dev) Send(data SendMailData) error {
	fmt.Println("====== SENDING EMAIL ======")
	fmt.Println("from: ", data.From)
	fmt.Println("ReplyTo: ", data.ReplyTo)
	fmt.Println("to: ", data.To)
	fmt.Println("subject: ", data.Subject)
	fmt.Printf("body\n%s\n\n", data.TextBody)
	fmt.Println("====== /SENDING EMAIL ======")
	return nil
}
