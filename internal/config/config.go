package config

import (
	"bytes"
	"fmt"
	"github.com/HighStakesSwitzerland/tendermint/config"
	"github.com/HighStakesSwitzerland/tendermint/libs/log"
	"github.com/HighStakesSwitzerland/tendermint/types"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
)

var (
	logger = log.MustNewDefaultLogger("text", "info", false)
)

// TSConfig extends tendermint P2PConfig with the things we need
type TSConfig struct {
	Chains []P2PConfig `mapstructure:"chains"`

	LogLevel string `mapstructure:"log_level"`
	HttpPort string `mapstructure:"http_port"`
}

type P2PConfig struct {
	config.Config `mapstructure:",squash"`
	ChainId       string `mapstructure:"chain_id"`
	PrettyName    string `mapstructure:"pretty_name"`
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

func InitConfigs() (*TSConfig, types.NodeKey) {
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
		os.Exit(0)
	} else {
		panic(err)
	}

	// only one node key for all chains
	nodeKeyFilePath := filepath.Join(homeDir, "node_key.json")
	nodeKey, err := types.LoadOrGenNodeKey(nodeKeyFilePath)
	if err != nil {
		panic(err)
	}

	logger.Info("Node key for all chains: ", "nodeId", nodeKey.ID)

	checkActiveChains(&tsConfig)

	return &tsConfig, nodeKey
}

func checkActiveChains(tsConfig *TSConfig) {
	// get field names of the config
	fieldNames := reflect.TypeOf(TSConfig{})
	names := make([]string, fieldNames.NumField())
	for i := range names {
		names[i] = fieldNames.Field(i).Name
	}

	// for each chain, check the config is ok
	value := reflect.Indirect(reflect.ValueOf(tsConfig))
	for i := 0; i < len(names); i++ {
		chain := value.FieldByName(names[i]).Interface()
		if reflect.TypeOf(chain) == reflect.TypeOf(P2PConfig{}) {
			chainCfg := chain.(P2PConfig)
			if chainCfg.ChainId == "" || chainCfg.P2P.BootstrapPeers == "" {
				logger.Info(fmt.Sprintf("%s config is incomplete, this chain will not be used", names[i]))
			} else {
				value.FieldByName(names[i]).FieldByName("Enable").SetBool(true)
			}
		}
	}
}

func initDefaultConfig() TSConfig {
	tsConfig := TSConfig{
		Chains:   []P2PConfig{*defaultP2PConfig(0)},
		LogLevel: "info",
		HttpPort: "8090",
	}
	return tsConfig
}

func defaultP2PConfig(port int) *P2PConfig {
	p := &P2PConfig{
		Config: *config.DefaultConfig(),
	}
	p.P2P.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%d", 26656+port)
	return p
}

// WriteConfigFile renders config using the template and writes it to configFilePath.
func writeConfigFile(configFilePath string, config *TSConfig) {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	_ = os.WriteFile(configFilePath, buffer.Bytes(), 0644)
}
