package mailer

import (
	"bytes"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
)

// Define a Mailer struct which contains a mail.Dialer instance (used to connect to a
// SMTP server) and the // want the email to be
type Mailer struct {
	dialer *mail.Dialer
	sender string
}

func New(host string, port int, username, password, sender string) Mailer {
	// Initialize a new mail.Dialer instance with the given SMTP server settings. We
	// also configure this to use a 5-second timeout whenever we send an email.
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second
	// Return a Mailer instance containing the dialer and sender information.
	return Mailer{dialer: dialer,
		sender: sender,
	}
}

// Define a Send() method on the Mailer type. This takes the recipient email address
// as the first parameter, the name of the file containing the templates, and any
// dynamic data for the templates as an interface{} parameter.
func (m Mailer) Send(recipient, templateFile string, data interface{}) error {
	tmpl, err := template.ParseFiles("internal/mailer/templates/" + templateFile)
	if err != nil {
		return err
	}
	subject := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&subject, "subject", data); err != nil {
		return err
	}
	plainBody := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&plainBody, "plainBody", data); err != nil {
		return err
	}
	htmlBody := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&htmlBody, "htmlBody", data); err != nil {
		return err
	}

	// Use the mail.NewMessage() function to initialize a new mail.Message instance.
	// Then we use the SetHeader() method to set the email recipient, sender and subject
	// headers, the SetBody() method to set the plain-text body, and the AddAlternative()
	// method to set the HTML body. It's important to note that AddAlternative() should // always be called *after* SetBody().
	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	// Try sending the email up to three times before aborting and returning the final
	// error. We sleep for 500 milliseconds between each attempt.
	for i := 1; i <= 3; i++ {
		err = m.dialer.DialAndSend(msg)
		// If everything worked, return nil.
		if nil == err {
			return nil
		}
		// If it didn't work, sleep for a short time and retry.
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}
