package seednode

import (
	"fmt"
	"github.com/HighStakesSwitzerland/tendermint/internals/p2p"
	"github.com/HighStakesSwitzerland/tendermint/internals/p2p/pex"
	"github.com/HighStakesSwitzerland/tendermint/libs/log"
	tmos "github.com/HighStakesSwitzerland/tendermint/libs/os"
	tmstrings "github.com/HighStakesSwitzerland/tendermint/libs/strings"
	"github.com/HighStakesSwitzerland/tendermint/types"
	"github.com/HighStakesSwitzerland/tendermint/version"
	"github.com/mitchellh/go-homedir"
	"path/filepath"
	"time"
)

var (
	logger = log.MustNewDefaultLogger("text", "info", false)
)

func StartSeedNodes(seedConfig *TSConfig, nodeKey *types.NodeKey) []p2p.Switch {
	var switches []p2p.Switch

	for _, chain := range seedConfig.Chains {
		if sw := startSeedNode(&chain, nodeKey); sw != nil {
			switches = append(switches, *sw)
		}
	}

	return switches
}

func startSeedNode(cfg *P2PConfig, nodeKey *types.NodeKey) *p2p.Switch {
	logger.Info("Starting Seed Node for chain " + cfg.ChainId)

	nodeInfo := types.NodeInfo{
		ProtocolVersion: types.ProtocolVersion{
			P2P:   version.P2PProtocol,
			Block: version.BlockProtocol,
			App:   0,
		},
		NodeID:     nodeKey.ID,
		Version:    "1.0.0",
		Network:    cfg.ChainId,
		ListenAddr: cfg.P2P.ListenAddress,
		Moniker:    fmt.Sprintf("%s-multiseed", cfg.ChainId),
		Channels:   []byte{byte(0x00)},
	}

	// set conn settings
	cfg.P2P.RecvRate = 5120000
	cfg.P2P.SendRate = 5120000
	cfg.P2P.MaxPacketMsgPayloadSize = 1024
	cfg.P2P.FlushThrottleTimeout = 100 * time.Millisecond
	cfg.P2P.AllowDuplicateIP = true
	cfg.P2P.DialTimeout = 30 * time.Second
	cfg.P2P.HandshakeTimeout = 20 * time.Second
	cfg.P2P.MaxNumInboundPeers = 2048

	userHomeDir, _ := homedir.Dir()
	addrBookFilePath := filepath.Join(userHomeDir, ".multiseed", "addrbook-"+cfg.ChainId+".json")
	addrBook := pex.NewAddrBook(addrBookFilePath, cfg.P2P.AddrBookStrict)

	pexReactor := pex.NewReactor(addrBook, &pex.ReactorConfig{
		SeedMode:                     true,
		Seeds:                        tmstrings.SplitAndTrim(cfg.P2P.BootstrapPeers, ",", " "),
		SeedDisconnectWaitPeriod:     5 * time.Minute, // default is 28 hours, we just want to harvest as many addresses as possible
		PersistentPeersMaxDialPeriod: 5 * time.Minute, // use exponential back-off
	})

	transport := p2p.NewMConnTransport(
		logger, p2p.MConnConfig(cfg.P2P), []*p2p.ChannelDescriptor{},
		p2p.MConnTransportOptions{
			MaxAcceptedConnections: uint32(cfg.P2P.MaxNumInboundPeers),
		},
	)

	addr, err := types.NewNetAddressString(
		nodeKey.ID.AddressString(nodeInfo.ListenAddr),
	)
	if err != nil {
		panic(err)
	}
	if err := transport.Listen(p2p.NewEndpoint(addr)); err != nil {
		panic(err)
	}
	sw := p2p.NewSwitch(cfg.P2P, transport)

	sw.SetLogger(log.MustNewDefaultLogger("text", "warn", false))
	sw.SetNodeKey(*nodeKey)
	sw.SetAddrBook(addrBook)
	sw.AddReactor("pex", pexReactor)

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
