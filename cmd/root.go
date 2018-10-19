package cmd

import (
	"io/ioutil"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"context"
	"time"
	"os/signal"

	"github.com/magiconair/properties"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/myopenfactory/client/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

var (
	cfgFile string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")

	rootCmd.Flags().StringP("username", "u", "", "username")
	viper.BindPFlag("username", rootCmd.Flags().Lookup("username"))

	rootCmd.Flags().StringP("password", "p", "", "password")
	viper.BindPFlag("password", rootCmd.Flags().Lookup("password"))

	rootCmd.Flags().String("url", "", "base url for the plattform")
	viper.BindPFlag("url", rootCmd.Flags().Lookup("url"))

	rootCmd.Flags().StringP("clientcert", "k", "", "client certificate (PEM)")
	viper.BindPFlag("clientcert", rootCmd.Flags().Lookup("clientcert"))

	rootCmd.Flags().StringP("cafile", "c", "", "ca file (PEM)")
	viper.BindPFlag("cafile", rootCmd.Flags().Lookup("cafile"))

	rootCmd.Flags().String("proxy", "", "proxy url")
	viper.BindPFlag("proxy", rootCmd.Flags().Lookup("proxy"))

	rootCmd.Flags().String("logLevel", "INFO", "log level")
	viper.BindPFlag("log.level", rootCmd.Flags().Lookup("logLevel"))

	rootCmd.Flags().String("logFolder", "", "folder for log files")
	viper.BindPFlag("log.folder", rootCmd.Flags().Lookup("logFolder"))

	rootCmd.Flags().String("logSyslog", "", "syslog server address")
	viper.BindPFlag("log.syslog", rootCmd.Flags().Lookup("logSyslog"))

	rootCmd.Flags().String("logEventlog", "", "eventlog name")
	viper.BindPFlag("log.eventlog", rootCmd.Flags().Lookup("logEventlog"))

	rootCmd.Flags().String("logMailHost", "", "mail server address")
	viper.BindPFlag("log.mail.host", rootCmd.Flags().Lookup("logMailHost"))

	rootCmd.Flags().Int("logMailPort", 25, "mail server port")
	viper.BindPFlag("log.mail.port", rootCmd.Flags().Lookup("logMailPort"))

	rootCmd.Flags().String("logMailFrom", "", "sender email address")
	viper.BindPFlag("log.mail.from", rootCmd.Flags().Lookup("logMailFrom"))

	rootCmd.Flags().String("logMailTo", "", "receiver email address")
	viper.BindPFlag("log.mail.to", rootCmd.Flags().Lookup("logMailTo"))

	rootCmd.Flags().String("logMailUsername", "", "mail server username")
	viper.BindPFlag("log.mail.username", rootCmd.Flags().Lookup("logMailUsername"))

	rootCmd.Flags().String("logMailPassword", "", "mail server password")
	viper.BindPFlag("log.mail.password", rootCmd.Flags().Lookup("logMailPassword"))

	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.SetEnvPrefix("client")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if cfgFile == "" {
		switch {
		case runtime.GOOS == "windows":
			cfgFile = os.Getenv("ProgramData") + "/myOpenFactory/client/config.properties"
		case runtime.GOOS == "linux":
			cfgFile = "/etc/myof-client/config.properties"
		}
	}

	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(cfgFile), 0); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		converted, err := convertYAML(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if !converted {
			f, err := os.Create(cfgFile)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			f.Close()
		}
	}

	fmt.Println(cfgFile)

	viper.SetConfigFile(cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Couldn't read config:", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "myof-client",
	Short: "myof-client is a very simple message acces for the myopenfactory platform",
Run: func(cmd *cobra.Command, args []string) {
		logLevel := viper.GetString("log.level")
		if logLevel != "" {
			log.WithLevel(logLevel)
		}

		logSyslog := viper.GetString("log.syslog")
		if logSyslog != "" {
			log.WithSyslog(logSyslog)
		}

		eventLog := viper.GetString("log.eventlog")
		if eventLog != "" {
			log.WithEventlog(eventLog)
		}

		logMailHost := viper.GetString("log.mail.host")
		if logMailHost != "" {
			addr := fmt.Sprintf("%s:%d", logMailHost, viper.GetInt("log.mail.port"))
			logMailFrom := viper.GetString("log.mail.from")
			logMailTo := viper.GetString("log.mail.to")
			logMailUsername := viper.GetString("log.mail.username")
			logMailPassword := viper.GetString("log.mail.password")
			log.WithMail("myOpenFactory Client", addr, logMailFrom, logMailTo, logMailUsername, logMailPassword)
		}

		logFolder := viper.GetString("log.folder")
		if logFolder != "" {
			log.WithFilesystem(logFolder)
		}

		log.Infof("Starting myOpenFactory client %v", version)

		cafile := viper.GetString("cafile")
		clientcert := viper.GetString("clientcert")

		if _, err := os.Stat(cafile); os.IsNotExist(err) {
			cafile = filepath.Join(filepath.Dir(cfgFile), cafile)
		}

		if _, err := os.Stat(clientcert); os.IsNotExist(err) {
			clientcert = filepath.Join(filepath.Dir(cfgFile), clientcert)
		}

		opts := []client.Option{
			client.WithUsername(viper.GetString("username")),
			client.WithPassword(viper.GetString("password")),
			client.WithURL(viper.GetString("url")),
			client.WithCA(cafile),
			client.WithCert(clientcert),
			client.WithProxy(viper.GetString("proxy")),
		}

		cl, err := client.New(fmt.Sprintf("Core_"+version), opts...)
		if err != nil {
			log.Errorf("error while creating client: %v", err)
			os.Exit(1)
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("recover client: %v", r)
				}
			}()
			if err := cl.Run(); err != nil {
				log.Errorf("error while running client: %v", err)
				os.Exit(1)
			}
		}()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)
		signal.Notify(stop, os.Kill)

		<-stop

		log.Infof("closing client")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cl.Shutdown(ctx)
		log.Infof("client gracefully stopped")
	},
}

// Execute execute the application with cobra
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// check if yaml file exists and convert		
func convertYAML(cfgFile string) (bool, error) {
	var file string

	base := filepath.Join(filepath.Dir(cfgFile), strings.TrimSuffix(filepath.Base(cfgFile), filepath.Ext(cfgFile)))

	if b, _ := exists(base+".yml"); b{
		file = base+".yml"
	}
	if b,_ := exists(base+".yaml"); b {
		file = base+".yaml"
	}
	
	if file == "" {
		return false, nil
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		return false, err
	}

	var c map[string]interface{}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return false, err
	}

	prop := properties.NewProperties()
	for key, value := range c {
		switch x := value.(type) {
		case string:
			prop.Set(key, x)
		case map[interface{}]interface{}:
			for nestedKey, nestedValue := range x {
				k, ok := nestedKey.(string)
				if !ok {
					continue
				}
				v, ok := nestedValue.(string)
				if !ok {
					continue
				}
				prop.Set(fmt.Sprintf("%s.%s", key, k), v)
			}
		default:
			fmt.Printf("%+T", x)
		}
	}

	f, err := os.Create(cfgFile)
	if err != nil {
		return false, fmt.Errorf("failed to open cofnig file %q: %v", cfgFile, err)
	}
	defer f.Close()

	if _, err := prop.Write(f, properties.UTF8); err != nil {
		return false, fmt.Errorf("failed to save config file: %q: %v", cfgFile, err)
	}

	// if err := os.Remove(file); err != nil {
	// 	return false, fmt.Errorf("failed to remove converted config: %q: %v", file, err)
	// }

	return true, nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}