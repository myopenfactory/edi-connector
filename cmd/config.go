package cmd

import (
	"github.com/myopenfactory/client/pkg/log"
	"fmt"
	"os"
	"strings"
	"path/filepath"
	"runtime"

	"github.com/magiconair/properties"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(configCmd)
	rootCmd.PersistentFlags().String("config", "", "config file")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	configCmd.AddCommand(listCmd)
	configCmd.AddCommand(getCmd)
	configCmd.AddCommand(setCmd)
}

func initConfig() {
	viper.SetEnvPrefix("client")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	cfgFile := viper.GetString("config")
	if cfgFile == "" {
		var cfgPath string
		switch {
		case runtime.GOOS == "windows":
			cfgPath = filepath.Join(os.Getenv("ProgramData"), "myOpenFactory", "client")
		case runtime.GOOS == "linux":
			cfgPath = filepath.Join("etc", "myof-client", "config")
		}
		cfgFile = filepath.Join(cfgPath, "config.properties")

		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			_, err := os.Create(cfgFile)
			if err != nil {
				fmt.Println("failed to create config file:", err)
				os.Exit(1)
			}
		}
	}

	viper.SetConfigFile(cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("failed to read config:", err)
		os.Exit(1)
	}

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

	log.Infof("Using config: %s", cfgFile)
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "manage the configuration",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all key value pairs",
	Run: func(cmd *cobra.Command, args []string) {
		cfgFile := viper.ConfigFileUsed()
		p, err := properties.LoadFile(cfgFile, properties.UTF8)
		if err != nil {
			fmt.Printf("failed to parse config file: %s", err)
			os.Exit(1)
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Key", "Value"})
		for _, key := range p.Keys() {
			table.Append([]string{key, p.GetString(key, "")})
		}
		table.Render()
	},
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "getting value from key",
	Run: func(cmd *cobra.Command, args []string) {
		cfgFile := viper.ConfigFileUsed()
		p, err := properties.LoadFile(cfgFile, properties.UTF8)
		if err != nil {
			fmt.Println("failed to parse config file: %s", err)
			os.Exit(1)
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Key", "Value"})
		for _, arg := range args {
			table.Append([]string{arg, p.GetString(arg, "")})
		}
		table.Render() // Send output
	},
}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "setting key",
	Run: func(cmd *cobra.Command, args []string) {
		cfgFile := viper.ConfigFileUsed()
		p, err := properties.LoadFile(cfgFile, properties.UTF8)
		if err != nil {
			fmt.Println("failed to load config file")
			os.Exit(1)
		}

		if len(args) % 2 != 0 {
			fmt.Println("uneven key-value-pairs")
			os.Exit(1)
		}

		for i := 0; i < len(args); i+=2 {
			_, _, err = p.Set(args[i], args[i+1])
			if err != nil {
				fmt.Printf("failed to set %q: %v\n", args[i], err)
				os.Exit(1)
			}
		}

		f, err := os.OpenFile(cfgFile, os.O_RDWR, 0)
		if err != nil {
			fmt.Printf("failed to open config file %q: %v", cfgFile, err)
			os.Exit(1)
		}
		defer f.Close()

		if _, err := p.Write(f, properties.UTF8); err != nil {
			fmt.Printf("failed to save config file: %q: %v", cfgFile, err)
			os.Exit(1)
		}
	},
}
