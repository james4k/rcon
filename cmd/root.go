package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	host     string
	password string
	command  string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rcon",
	Short: "Basic RCON CLI interface",
	Long:  `Basic RCON CLI interface`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		if host == "" {
			log.Fatalf("host cannot be empty")
		}
		if password == "" {
			log.Fatalf("password cannot be empty")
		}
		conn, err := rcon.Dial(ctx, host, password, 10*time.Second)
		if err != nil {
			log.Fatalf("Failed to dial server")
		}
		// Exec single command and return
		if command != "" {
			resp, err := conn.Exec(command)
			if err != nil {
				log.Fatalf("Failed to exec command: %v", err)
			}
			fmt.Printf("%s\n", resp)
			return
		}
		// REPL CLI
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("rcon> ")
			cIn, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Failed to read line: %v", err)
			}
			c := strings.ToLower(strings.Trim(cIn, " \n"))
			if c == "quit" || c == "exit" {
				log.Printf("Exiting (user initiated)")
				return
			}
			resp, err := conn.Exec(c)
			if err != nil {
				log.Fatalf("Failed to exec command: %v", err)
			}
			fmt.Printf("%s", resp)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rcon.yaml)")
	rootCmd.PersistentFlags().StringVar(&host, "host", "localhost:27015", "config file (default: localhost:27015)")
	rootCmd.PersistentFlags().StringVar(&password, "password", "localhost:27015", "RCON password (default: '')")
	rootCmd.PersistentFlags().StringVar(&command, "command", "", "Command to run (default: status)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".rcon" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".rcon")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
