package mailer

import (
	"bytes"
	"embed"
	"github.com/wneessen/go-mail"
	"html/template"
	"time"
)

// templateFS is a variable with the type embed.FS to hold
// the email templates inside the templates directory.
//
//go:embed templates
var templateFS embed.FS

// Mailer is a struct which contains mail.Client instance(used to connect and interact with an SMTP server)
// sender information for emails (the name and address you want the email to be from,
// such as "Alice Smith <alicesmith@example.com>".) and also methods for sending emails.
type Mailer struct {
	client *mail.Client
	sender string
}

// New returns a new Mailer with configured mail.Client and sender information.
func New(host string, port int, sender, username, password string) (*Mailer, error) {
	client, err := mail.NewClient(
		host,
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithSSL(),
		mail.WithPort(port),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTimeout(10*time.Second),
	)

	if err != nil {
		return nil, err
	}

	return &Mailer{client: client, sender: sender}, nil
}

// Send is a method that send an email, with the provided template to the recipient.
func (m *Mailer) Send(recipient, templateName string, data any) error {
	// Use the ParseFS() to parse the required template file from
	// the embedded file system.
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateName)
	if err != nil {
		return err
	}

	// Execute the named template "subject", passing in the dynamic data and storing
	// the result in a bytes.Buffer.
	var subject bytes.Buffer
	if err = tmpl.ExecuteTemplate(&subject, "subject", data); err != nil {
		return err
	}

	// Execute the named template "plainBody", passing in the dynamic data and storing
	// the result in a bytes.Buffer.
	var plainBody bytes.Buffer
	if err = tmpl.ExecuteTemplate(&plainBody, "plainBody", data); err != nil {
		return err
	}

	// Execute the named template "htmlBody", passing in the dynamic data and storing
	// the result in a bytes.Buffer.
	var htmlBody bytes.Buffer
	if err = tmpl.ExecuteTemplate(&htmlBody, "htmlBody", data); err != nil {
		return err
	}

	// Use the mail.NewMsg() to initialize a new mail.Message instance.
	// Then set the subject, address "TO" and "FROM", and also the plain-text body
	// and html body alternative.
	msg := mail.NewMsg()
	msg.SetGenHeader(mail.HeaderSubject, subject.String())
	if err = msg.SetAddrHeader(mail.HeaderTo, recipient); err != nil {
		return err
	}
	if err = msg.SetAddrHeader(mail.HeaderFrom, m.sender); err != nil {
		return err
	}
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())

	// Send the message.
	return m.client.DialAndSend(msg)
}
