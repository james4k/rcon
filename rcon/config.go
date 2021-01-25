package rcon

import (
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var Config *rootConfig

type rootConfig struct {
	DefaultServers []string           `mapstructure:"default_servers"`
	DefaultCommand string             `mapstructure:"default_command"`
	Servers        map[string]*Server `mapstructure:"servers"`
}

type Server struct {
	Name     string `mapstructure:"name"`
	Host     string `mapstructure:"host"`
	Password string `mapstructure:"password"`
}

// Read reads in config file and ENV variables if set.
func ReadConfig(cfgFile string) error {
	// Find home directory.
	home, _ := homedir.Dir()
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.AddConfigPath("../")
	viper.SetConfigName("rcon")
	if os.Getenv("RCON_CONFIG") != "" {
		viper.SetConfigFile(os.Getenv("RCON_CONFIG"))
	} else if cfgFile != "" {
		viper.SetConfigName(cfgFile)
	}
	viper.AutomaticEnv() // read in environment variables that match
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
		cfg := &rootConfig{}
		if err := viper.Unmarshal(cfg); err != nil {
			return errors.Wrapf(err, "Failed to parse config")
		}
		for _, server := range cfg.Servers {
			if !strings.Contains(server.Host, ":") {
				server.Host = server.Host + ":27015"
			}
		}
		Config = cfg
		return nil
	} else {
		log.Errorf("Failed to read config: %v", err)
	}
	return nil
}
