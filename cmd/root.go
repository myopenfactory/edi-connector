package cmd

import (
	"fmt"
	"os"
	"context"
	"time"
	"os/signal"

	"github.com/myopenfactory/client/pkg/log"
	"github.com/myopenfactory/client/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "myof-client",
	Short: "myof-client is a very simple message acces for the myopenfactory platform",
Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting myOpenFactory client %v", Version)

		opts := []client.Option{
			client.WithUsername(viper.GetString("username")),
			client.WithPassword(viper.GetString("password")),
			client.WithURL(viper.GetString("url")),
			client.WithCA(viper.GetString("cafile")),
			client.WithCert(viper.GetString("clientcert")),
		}
		os.Setenv("HTTP_PROXY", viper.GetString("proxy"))

		cl, err := client.New(fmt.Sprintf("Core_"+Version), opts...)
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