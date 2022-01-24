package seednode

import (
	"bytes"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/config"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/p2p"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TSConfig extends tendermint P2PConfig with the things we need
type TSConfig struct {
	Terra      P2PConfig `mapstructure:"terra"`
	Sentinel   P2PConfig `mapstructure:"sentinel"`
	Persitence P2PConfig `mapstructure:"persistence"`
	Lum        P2PConfig `mapstructure:"lum"`
	Desmos     P2PConfig `mapstructure:"desmos"`
	Injective  P2PConfig `mapstructure:"injective"`
	Band       P2PConfig `mapstructure:"band"`
	Certik     P2PConfig `mapstructure:"certik"`
	Fetchai    P2PConfig `mapstructure:"fetchai"`
	Irisnet    P2PConfig `mapstructure:"irisnet"`
	Sifchain   P2PConfig `mapstructure:"sifchain"`

	LogLevel string `mapstructure:"log_level"`
	HttpPort int    `mapstructure:"http_port"`
}

type P2PConfig struct {
	config.P2PConfig `mapstructure:",squash"`
	ChainId          string `mapstructure:"chain_id"`
	Enable           bool
}

var configTemplate *template.Template

func init() {
	var err error
	tmpl := template.New("configFileTemplate").Funcs(template.FuncMap{
		"StringsJoin": strings.Join,
	})
	if configTemplate, err = tmpl.Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

func InitConfigs() (TSConfig, p2p.NodeKey) {
	var tsConfig TSConfig

	userHomeDir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	// init config directory & files if they don't exists yet
	homeDir := filepath.Join(userHomeDir, ".multiseed")
	if err = os.MkdirAll(homeDir, os.ModePerm); err != nil {
		panic(err)
	}

	configFilePath := filepath.Join(homeDir, "config.toml")
	viper.SetConfigName("config")
	viper.AddConfigPath(homeDir)

	if err := viper.ReadInConfig(); err == nil {
		logger.Info(fmt.Sprintf("Loading config file: %s", viper.ConfigFileUsed()))
		err := viper.Unmarshal(&tsConfig)
		if err != nil {
			panic("Invalid config file!")
		}
	} else if _, ok := err.(viper.ConfigFileNotFoundError); ok { // ignore not found error, return other errors
		logger.Info("No existing configuration found, generating one")
		tsConfig = initDefaultConfig()
		writeConfigFile(configFilePath, &tsConfig)
	} else {
		panic(err)
	}

	// only one node key for all chains
	nodeKeyFilePath := filepath.Join(homeDir, "node_key.json")
	nodeKey, err := p2p.LoadOrGenNodeKey(nodeKeyFilePath)
	if err != nil {
		panic(err)
	}

	logger.Info("Node key for all chains: ", "nodeId", nodeKey.ID())

	if tsConfig.Terra.Seeds == "" || tsConfig.Terra.ChainId == "" {
		logger.Info("No ChainId or Seeds for config [terra]; this chain will not be used")
		tsConfig.Terra.Enable = false
	} else {
		tsConfig.Terra.Enable = true
	}
	if tsConfig.Band.Seeds == "" || tsConfig.Band.ChainId == "" {
		logger.Info("No ChainId or Seeds for config [band]; this chain will not be used")
		tsConfig.Band.Enable = false
	} else {
		tsConfig.Band.Enable = true
	}

	return tsConfig, *nodeKey
}

func initDefaultConfig() TSConfig {
	tsConfig := TSConfig{
		Terra:      *defaultP2PConfig(),
		Sentinel:   *defaultP2PConfig(),
		Persitence: *defaultP2PConfig(),
		Lum:        *defaultP2PConfig(),
		Desmos:     *defaultP2PConfig(),
		Injective:  *defaultP2PConfig(),
		Band:       *defaultP2PConfig(),
		Certik:     *defaultP2PConfig(),
		Fetchai:    *defaultP2PConfig(),
		Irisnet:    *defaultP2PConfig(),
		Sifchain:   *defaultP2PConfig(),
		LogLevel:   "info",
		HttpPort:   8090,
	}
	return tsConfig
}

func defaultP2PConfig() *P2PConfig {
	p := &P2PConfig{
		P2PConfig: *config.DefaultP2PConfig(),
		ChainId:   "",
		Enable:    false,
	}
	p.ListenAddress = "tcp://127.0.0.1:26656"
	return p
}

// WriteConfigFile renders config using the template and writes it to configFilePath.
func writeConfigFile(configFilePath string, config *TSConfig) {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	tmos.MustWriteFile(configFilePath, buffer.Bytes(), 0644)
}

const defaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

# NOTE: Any path below can be absolute (e.g. "/var/myawesomeapp/data") or
# relative to the home directory (e.g. "data"). The home directory is
# "$HOME/.tendermint" by default, but could be changed via $TMHOME env variable
# or --home cmd flag.

#######################################################
###     Multiseed Server Configuration Options      ###
#######################################################
# Port for the frontend
http_port = "{{ .HttpPort }}"

# Output level for logging: "info" or "debug". debug will enable pex and addrbook verbose logs
log_level = "{{ .LogLevel }}"

laddr = "{{ .ListenAddress }}"

# Chain specific config
[terra]
chain_id = "{{ .Terra.ChainId }}"
seeds = "{{ .Terra.Seeds }}"

[band]
chain_id = "{{ .Band.ChainId }}"
seeds = "{{ .Band.Seeds }}"

`
