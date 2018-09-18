package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.client.yaml)")
}

func initConfig() {
	viper.SetEnvPrefix("client")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Don't forget to read config either from cfgFile or from home directory!
	switch {
	case cfgFile != "":
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	case runtime.GOOS == "windows":
		// Search config in home directory with name ".client" (without extension).
		viper.AddConfigPath(os.Getenv("ProgramData") + "/myOpenFactory/client/")
		viper.SetConfigName("config")
		cfgFile = os.Getenv("ProgramData") + "/myOpenFactory/client/config.properties"
	default:
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".client" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".myofclient")
		cfgFile = home + ".myofclient.properties"
	}

	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(cfgFile), 0); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		f, err := os.Create(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		f.Close()
	}

	log.Infof("Using configuration file %q", cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}

}

var rootCmd = &cobra.Command{
	Use:   "myof-client",
	Short: "myof-client is a very simple message acces for the myopenfactory platform",
}

// Execute execute the application with cobra
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
