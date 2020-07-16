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
)

var (
	host     string
	password string
)

// rootCmd represents the base command when called without any sub commands
var rootCmd = &cobra.Command{
	Use:     "rcon [command]",
	Short:   "Basic RCON CLI interface",
	Long:    `Basic RCON CLI interface`,
	Version: rcon.BuildVersion,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		command := strings.Join(args, " ")
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&host, "host", "H", "localhost:27015",
		"Remote host, host:port format")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "RCON password")
}
