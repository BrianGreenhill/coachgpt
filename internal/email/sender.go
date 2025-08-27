package email

import "log"

type Sender interface {
	Send(to, subject, html string) error
}

type StdoutSender struct{}

func (StdoutSender) Send(to, subject, html string) error {
	log.Printf("EMAIL to=%s subject=%s\n%s", to, subject, html)
	return nil
}
