package seednode

import (
	"bytes"
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/config"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/p2p"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
)

// TSConfig extends tendermint P2PConfig with the things we need
type TSConfig struct {
	Osmosis       P2PConfig `mapstructure:"osmosis"`
	Terra         P2PConfig `mapstructure:"terra"`
	Bombay        P2PConfig `mapstructure:"bombay"`
	Sentinel      P2PConfig `mapstructure:"sentinel"`
	Persistence   P2PConfig `mapstructure:"persistence"`
	Lum           P2PConfig `mapstructure:"lum"`
	Desmos        P2PConfig `mapstructure:"desmos"`
	Injective     P2PConfig `mapstructure:"injective"`
	Band          P2PConfig `mapstructure:"band"`
	Certik        P2PConfig `mapstructure:"certik"`
	Fetchai       P2PConfig `mapstructure:"fetchai"`
	Irisnet       P2PConfig `mapstructure:"irisnet"`
	Sifchain      P2PConfig `mapstructure:"sifchain"`
	Rizon         P2PConfig `mapstructure:"rizon"`
	Konstellation P2PConfig `mapstructure:"konstellation"`
	Provenance    P2PConfig `mapstructure:"provenance"`

	LogLevel string `mapstructure:"log_level"`
	HttpPort string `mapstructure:"http_port"`
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

func InitConfigs() (*TSConfig, *p2p.NodeKey) {
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
	nodeKey, err := p2p.LoadOrGenNodeKey(nodeKeyFilePath)
	if err != nil {
		panic(err)
	}

	logger.Info("Node key for all chains: ", "nodeId", nodeKey.ID())

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
			if chainCfg.Seeds == "" || chainCfg.ChainId == "" {
				logger.Info(fmt.Sprintf("%s config is incomplete, this chain will not be used", names[i]))
			} else {
				value.FieldByName(names[i]).FieldByName("Enable").SetBool(true)
			}
		}
	}
}

func initDefaultConfig() TSConfig {
	tsConfig := TSConfig{
		Terra:         *defaultP2PConfig(0),
		Band:          *defaultP2PConfig(1),
		Fetchai:       *defaultP2PConfig(2),
		Injective:     *defaultP2PConfig(3),
		Persistence:   *defaultP2PConfig(4),
		Irisnet:       *defaultP2PConfig(5),
		Sentinel:      *defaultP2PConfig(6),
		Certik:        *defaultP2PConfig(7),
		Lum:           *defaultP2PConfig(8),
		Sifchain:      *defaultP2PConfig(9),
		Desmos:        *defaultP2PConfig(10),
		Bombay:        *defaultP2PConfig(11),
		Rizon:         *defaultP2PConfig(12),
		Konstellation: *defaultP2PConfig(13),
		Provenance:    *defaultP2PConfig(14),

		LogLevel: "info",
		HttpPort: "8090",
	}
	return tsConfig
}

func defaultP2PConfig(port int) *P2PConfig {
	p := &P2PConfig{
		P2PConfig: *config.DefaultP2PConfig(),
		ChainId:   "",
		Enable:    false,
	}
	p.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%d", 26656+port)
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
