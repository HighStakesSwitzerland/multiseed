package seednode

import (
	"fmt"
	"github.com/HighStakesSwitzerland/tendermint/config"
	"github.com/HighStakesSwitzerland/tendermint/libs/log"
	tmos "github.com/HighStakesSwitzerland/tendermint/libs/os"
	"github.com/HighStakesSwitzerland/tendermint/node"
	"github.com/HighStakesSwitzerland/tendermint/proto/tendermint/p2p"
	"github.com/HighStakesSwitzerland/tendermint/types"
	"github.com/HighStakesSwitzerland/tendermint/version"
	"reflect"
	"time"
)

var (
	logger = log.MustNewDefaultLogger("text", "info", false)
)

func StartSeedNodes(seedConfig *TSConfig, nodeKey *types.NodeKey) {
	var switches []p2p.Switch

	value := reflect.ValueOf(seedConfig)
	for i := 0; i < reflect.Indirect(value).NumField(); i++ {
		chain := reflect.Indirect(value).Field(i).Interface()
		if reflect.TypeOf(chain) == reflect.TypeOf(P2PConfig{}) {
			chainCfg := chain.(P2PConfig)
			if sw := startSeedNode(&chainCfg, nodeKey, seedConfig.LogLevel); sw != nil {
				switches = append(switches, *sw)
			}
		}
	}

	return switches
}

func startSeedNode(cfg *P2PConfig, nodeKey *p2p.NodeInfo, configLogLevel string) *p2p.Switch {
	if cfg.Enable == false {
		return nil
	}

	logger.Info("Starting Seed Node for chain " + cfg.ChainId)

	id, _ := types.NewNodeID(nodeKey.NodeID)

	// NodeInfo gets info on your node
	nodeInfo := types.NodeInfo{
		ProtocolVersion: types.ProtocolVersion{
			P2P:   version.P2PProtocol,
			Block: version.BlockProtocol,
			App:   0,
		},
		NodeID:     id,
		ListenAddr: cfg.P2P.ListenAddress,
		Moniker:    fmt.Sprintf("%s-multiseed", cfg.ChainId),
		Version:    "1.0.0",
		Network:    cfg.ChainId,
		Channels:   []byte{byte(0x00)},
	}

	addr, err := types.NewNetAddressString(id.AddressString(cfg.P2P.ListenAddress))
	if err != nil {
		panic(err)
	}

	// set conn settings
	cfg.P2P.RecvRate = 51200
	cfg.P2P.SendRate = 51200
	cfg.P2P.MaxPacketMsgPayloadSize = 1024
	cfg.P2P.FlushThrottleTimeout = 100 * time.Second
	cfg.P2P.AllowDuplicateIP = true
	cfg.P2P.DialTimeout = 5 * time.Second
	cfg.P2P.HandshakeTimeout = 3 * time.Second
	cfg.P2P.PexReactor = true
	cfg.Mode = config.ModeSeed
	cfg.NodeKey = ".multiseed"
	cfg.P2P.AddrBook = "addrbook-" + cfg.ChainId + ".json"

	seedNode, err := node.NewDefault(&cfg.Config, logger)
	if err != nil {
		panic(err)
	}

	seedNode.Start()

	sw := p2p.NewSwitch(&cfg.P2PConfig, transport)

	sw.SetNodeKey(nodeKey)
	sw.SetAddrBook(addrBook)
	sw.AddReactor("pex", pexReactor)

	sw.SetLogger(configuredLogger.With("module", "switch"))
	addrBook.SetLogger(configuredLogger.With("module", "addrbook", "chain", cfg.ChainId))
	pexReactor.SetLogger(configuredLogger.With("module", "pex"))

	// last
	sw.SetNodeInfo(nodeInfo)

	err = sw.Start()
	if err != nil {
		panic(err)
	}

	tmos.TrapSignal(logger, func() {
		logger.Info("shutting down addrbooks...")
		_ = addrBook.Stop()
		_ = sw.Stop()
	})

	return sw
}
