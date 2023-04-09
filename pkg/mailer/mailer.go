package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
)

//go:embed "templates"
var templateFS embed.FS

type Mailer struct {
	dialer *mail.Dialer
	sender string
}

func New(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

func (m Mailer) Send(recipient, templateFile string, data any) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// no se suele usar la funcion new, se suele usar el operador & para crear un puntero
	// yo lo veo como que tiene 2 ventajas
	// 1 - es mas claro que se esta creando un puntero
	// 2 - es mas explicito que estamos con el valor cero del struct, y no nos confundimos con un
	//        new de otro lenguaje que puede tener un comportamiento distinto
	// ademas a nivel del compilador, un new le estas diciendo que aloque memoria en el heap
	// mientras que el & seguramente entre en el stack que es sustancialmente mas rapido
	subject := &bytes.Buffer{}
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainbody := &bytes.Buffer{}
	err = tmpl.ExecuteTemplate(plainbody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlbody := &bytes.Buffer{}
	err = tmpl.ExecuteTemplate(htmlbody, "htmlBody", data)
	if err != nil {
		return err
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainbody.String())
	msg.AddAlternative("text/html", htmlbody.String())

	err = m.dialer.DialAndSend(msg)
	if err != nil {
		return err
	}

	return nil
}
