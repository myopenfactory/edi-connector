package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func LogLevel() string {
	return viper.GetString("log.level")
}

func LogFile() string {
	return viper.GetString("log.folder")
}

func LogSyslogAddress() string {
	return viper.GetString("log.syslog")
}

func LogFolder() string {
	return viper.GetString("log.folder")
}

func LogMailHost() string {
	return viper.GetString("log.mail.host")
}

func LogMailPort() int {
	return viper.GetInt("log.mail.port")
}

func LogMailAddress() string {
	host := LogMailHost()
	port := LogMailPort()
	if host == "" || port == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func LogMailSender() string {
	return viper.GetString("log.mail.from")
}

func LogMailReceiver() string {
	return viper.GetString("log.mail.to")
}

func LogMailUsername() string {
	return viper.GetString("log.mail.username")
}

func LogMailPassword() string {
	return viper.GetString("log.mail.password")
}