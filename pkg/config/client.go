package config

import (
	"github.com/spf13/viper"
)

func Username() string {
	return viper.GetString("username")
}

func Password() string {
	return viper.GetString("password")
}

func URL() string {
	return viper.GetString("url")
}

func ClientCertificate() string {
	return viper.GetString("clientcert")
}

func CertificateAuthority() string {
	return viper.GetString("cafile")
}

func Proxy() string {
	return viper.GetString("proxy")
}