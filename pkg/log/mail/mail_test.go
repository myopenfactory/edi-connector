package mail

import (
	"net/smtp"
	"reflect"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestMailHook_Levels(t *testing.T) {
	tests := []struct {
		name string
		hook *MailHook
		want []logrus.Level
	}{
		{
			name: "Testing",
			want: []logrus.Level{
				logrus.PanicLevel,
				logrus.FatalLevel,
				logrus.ErrorLevel,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.hook.Levels(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MailHook.Levels() = %v, want %v", got, tt.want)
			}
		})
	}
}

type emailRecorder struct {
	addr string
	auth smtp.Auth
	from string
	to   []string
	msg  []byte
}

func mockSend(errToReturn error) (func(string, smtp.Auth, string, []string, []byte) error, *emailRecorder) {
	r := new(emailRecorder)
	return func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		*r = emailRecorder{addr, a, from, to, msg}
		return errToReturn
	}, r
}

func TestMailHook_Fire(t *testing.T) {
	f, r := mockSend(nil)
	mh := &MailHook{
		AppName:  "TestApp",
		Address:  "localhost:1026",
		From:     "test@localhost",
		To:       "test@localhost",
		Username: "username",
		Password: "pw",
		send:     f,
	}

	err := mh.Fire(&logrus.Entry{
		Message: "TestMail",
		Data: logrus.Fields{
			"key": "value",
		},
		Time:  time.Date(2006, 1, 2, 15, 04, 05, 0, time.UTC),
		Level: logrus.ErrorLevel,
	})

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	msg := "Subject: TestApp-error\r\n\r\n2006-01-02T15:04:05Z-TestMail\r\n\r\n{\n\t\"key\": \"value\"\n}"
	if string(r.msg) != msg {
		t.Errorf("wrong message body: want = %x, got = %x", []byte(msg), r.msg)
	}

	err = mh.Fire(nil)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestNewMailHook(t *testing.T) {
	want := &MailHook{
		AppName:  "TestApp",
		Address:  "localhost:1027",
		From:     "sender@test.com",
		To:       "receipt@test.com",
		Username: "user",
		Password: "password",
	}

	got := New(want.AppName, want.Address, want.From, want.To, want.Username, want.Password)
	if got.AppName != want.AppName ||
		got.Address != want.Address ||
		got.From != want.From ||
		got.To != want.To ||
		got.Username != want.Username ||
		got.Password != want.Password {
		t.Errorf("NewMailHook() = %v, want %v", got, want)
	}
}
