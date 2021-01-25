package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/leighmacdonald/rcon/rcon"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"os"
	"strings"
	"time"
)

var (
	host     string
	password string
	env      string
	cfgFile  string
)

// rootCmd represents the base command when called without any sub commands
var rootCmd = &cobra.Command{
	Use:     "rcon [command]",
	Short:   "Basic RCON CLI interface",
	Long:    `Basic RCON CLI interface`,
	Version: rcon.BuildVersion,
	Args:    cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var envs []*rcon.Server
		if env != "" {
			for _, name := range strings.Split(env, ",") {
				sc, found := rcon.Config.Servers[name]
				if !found {
					log.Fatalf("Invalid server name: %s", name)
				}
				sc.Name = name
				envs = append(envs, sc)
			}
		}
		// Using CLI args for env config
		if len(envs) > 0 && (password != "" || host != "") {
			log.Warnf("Host and Password options are ignored when using env config")
		}
		if len(envs) == 0 {
			for _, name := range rcon.Config.DefaultServers {
				sc, found := rcon.Config.Servers[name]
				if !found {
					log.Fatalf("Invalid default server name: %s", name)
				}
				sc.Name = name
				envs = append(envs, sc)
			}
		}
		if host != "" || password != "" {
			if host == "" {
				log.Fatalf("Host cannot be empty")
			}
			if password == "" {
				log.Fatalf("Password cannot be empty")
			}
			if !strings.Contains(host, ":") {
				host = host + ":27015"
			}
			envs = []*rcon.Server{{
				Name:     "default",
				Host:     host,
				Password: password,
			}}
		}
		command := strings.Join(args, " ")
		if command == "" && rcon.Config.DefaultCommand != "" {
			command = rcon.Config.DefaultCommand
		}
		ctx := context.Background()
		type serverState struct {
			config *rcon.Server
			conn   *rcon.RemoteConsole
		}
		var servers []serverState
		for _, sc := range envs {
			conn, err := rcon.Dial(ctx, sc.Host, sc.Password, 10*time.Second)
			if err != nil {
				log.Fatalf("Failed to dial server")
			}
			servers = append(servers, serverState{config: sc, conn: conn})
		}
		defer func() {
			for _, server := range servers {
				if err := server.conn.Close(); err != nil {
					log.Errorf("Failed to close connection properly: %v", err)
				}
			}
		}()
		if command != "" {
			for _, server := range servers {
				// Exec single command and exit
				if command != "" {
					resp, err := server.conn.Exec(command)
					if err != nil {
						log.Fatalf("Failed to exec command: %v", err)
					}
					fmt.Printf("[%s] %s\n", server.config.Name, resp)
				}
			}
			os.Exit(0)
		}
		// REPL CLI
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("rcon (%d hosts)> ", len(servers))
			cIn, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Print("\b")
					os.Exit(0)
				}
				log.Fatalf("Failed to read line: %v", err)
			}
			c := strings.ToLower(strings.Trim(cIn, " \n"))
			if c == "quit" || c == "exit" {
				log.Printf("Exiting (user initiated)")
				return
			}
			for _, server := range servers {
				resp, err := server.conn.Exec(c)
				if err != nil {
					log.Fatalf("Failed to exec command: %v", err)
				}
				fmt.Printf("%s", resp)
			}
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
	cobra.OnInitialize(func() {
		if err := rcon.ReadConfig(cfgFile); err != nil {
			log.Fatalf("Could not load & parse config: %v", err)
		}
	})
	rootCmd.PersistentFlags().StringVarP(&env, "env", "e", "", "Server environment to load")
	rootCmd.PersistentFlags().StringVarP(&host, "host", "H", "",
		"Remote host, host:port format")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "RCON password")

	var err error
	err = viper.BindPFlag("env", rootCmd.PersistentFlags().Lookup("env"))
	err = viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("host"))
	err = viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))
	if err != nil {
		log.Fatalf("Failed to bind config flags: %v", err)
	}
}
