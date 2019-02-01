package mail

import (
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EmailSender declares a method which transmits an email.
type EmailSender interface {
	Send(to []string, body []byte) error
}

// MailHook to sends logs by email with authentication.
type MailHook struct {
	AppName  string
	Address  string
	From     string
	To       string
	Username string
	Password string
	send     func(string, smtp.Auth, string, []string, []byte) error
}

// NewMailHook creates a MailHook and configures it from parameters.
func New(appname, address, sender, receiver, username, password string) *MailHook {
	return &MailHook{
		AppName:  appname,
		Address:  address,
		From:     sender,
		To:       receiver,
		Username: username,
		Password: password,
		send:     smtp.SendMail,
	}
}

// Fire is called when a log event is fired.
func (h *MailHook) Fire(entry *logrus.Entry) error {
	if entry == nil {
		return nil
	}

	body := fmt.Sprintf("%s-%s", entry.Time.Format(time.RFC3339Nano), entry.Message)
	subject := fmt.Sprintf("%s-%s", h.AppName, entry.Level)
	fields, _ := json.MarshalIndent(entry.Data, "", "\t")
	message := fmt.Sprintf("Subject: %s\r\n\r\n%s\r\n\r\n%s", subject, body, fields)

	receivers := strings.Split(h.To, ";")
	err := h.sendMail(receivers, []byte(message))
	return errors.Wrapf(err, "failed sending log mail")
}

func (h *MailHook) sendMail(to []string, body []byte) error {
	var auth smtp.Auth
	if h.Username != "" {
		host, _, err := net.SplitHostPort(h.Address)
		if err != nil {
			return errors.Wrap(err, "failed to split host port for smtp auth")
		}
		auth = smtp.PlainAuth("", h.Username, h.Password, host)
	}
	return h.send(h.Address, auth, h.From, to, body)
}

// Levels returns the available logging levels.
func (h *MailHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
	}
}
