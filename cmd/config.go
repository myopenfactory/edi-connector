package cmd

import (
	"fmt"
	"os"

	"github.com/magiconair/properties"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	configKey   string
	configValue string
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(listCmd)
	configCmd.AddCommand(getCmd)
	getCmd.Flags().StringVar(&configKey, "key", "", "properties key")
	getCmd.MarkFlagRequired("key")

	configCmd.AddCommand(setCmd)
	setCmd.Flags().StringVar(&configKey, "key", "", "properties key")
	setCmd.Flags().StringVar(&configValue, "value", "", "properties value")
	setCmd.MarkFlagRequired("key")
	setCmd.MarkFlagRequired("value")
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "manage the configuration",
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all key value pairs",
	Run: func(cmd *cobra.Command, args []string) {
		if cfgFile == "" {
			return
		}
		p, err := properties.LoadFile(cfgFile, properties.UTF8)
		if err != nil {
			fmt.Println("failed to load config file")
			os.Exit(1)
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Key", "Value"})
		for key, value := range p.Map() {
			table.Append([]string{key, value})
		}
		table.Render() // Send output
	},
}

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "getting value from key",
	Run: func(cmd *cobra.Command, args []string) {
		p, err := properties.LoadFile(cfgFile, properties.UTF8)
		if err != nil {
			fmt.Println("failed to load config file")
			os.Exit(1)
		}
		value, ok := p.Get(configKey)
		if !ok {
			fmt.Println("not found")
			os.Exit(1)
		}
		fmt.Println(value)
	},
}

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "setting key",
	Run: func(cmd *cobra.Command, args []string) {
		p, err := properties.LoadFile(cfgFile, properties.UTF8)
		if err != nil {
			fmt.Println("failed to load config file")
			os.Exit(1)
		}
		_, _, err = p.Set(configKey, configValue)
		if err != nil {
			fmt.Printf("failed to set %q: %v\n", configKey, err)
			os.Exit(1)
		}
		f, err := os.OpenFile(cfgFile, os.O_RDWR, 0)
		if err != nil {
			fmt.Printf("failed to open config file %q: %v", configKey, err)
			os.Exit(1)
		}
		defer f.Close()

		if _, err := p.Write(f, properties.UTF8); err != nil {
			fmt.Printf("failed to save config file: %q: %v", configKey, err)
			os.Exit(1)
		}
	},
}
