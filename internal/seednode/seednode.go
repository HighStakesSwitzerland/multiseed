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
	"github.com/highstakesswitzerland/multiseed/internal/config"
	"github.com/mitchellh/go-homedir"
	"path/filepath"
	"time"
)

var (
	logger     = log.MustNewDefaultLogger("text", "info", false)
	noOpLogger = log.NewNopLogger()
)

type SeedNodeConfig struct {
	Sw       *p2p.Switch
	Cfg      *config.P2PConfig
	AddrBook pex.AddrBook
}

func StartSeedNodes(seedConfig *config.TSConfig, nodeKey *types.NodeKey) []SeedNodeConfig {
	var seedNodes []SeedNodeConfig

	for i := 0; i < len(seedConfig.ChainConfigs); i++ {
		if sw, addrBook := startSeedNode(&seedConfig.ChainConfigs[i], nodeKey); sw != nil {
			seedNodes = append(seedNodes, SeedNodeConfig{sw, &seedConfig.ChainConfigs[i], addrBook})
		}
	}
	return seedNodes
}

func startSeedNode(cfg *config.P2PConfig, nodeKey *types.NodeKey) (*p2p.Switch, pex.AddrBook) {
	logger.Info(fmt.Sprintf("Starting Seed Node for chain %s [%s]", cfg.PrettyName, cfg.ChainId))

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
	cfg.P2P.RecvRate = 512000
	cfg.P2P.SendRate = 512000
	cfg.P2P.MaxPacketMsgPayloadSize = 1024
	cfg.P2P.FlushThrottleTimeout = 120 * time.Second
	cfg.P2P.AllowDuplicateIP = true
	cfg.P2P.DialTimeout = 30 * time.Second
	cfg.P2P.HandshakeTimeout = 20 * time.Second
	cfg.P2P.MaxNumInboundPeers = 4096

	userHomeDir, _ := homedir.Dir()
	addrBookFilePath := filepath.Join(userHomeDir, ".multiseed", "addrbook-"+cfg.ChainId+".json")
	addrBook := pex.NewAddrBook(addrBookFilePath, cfg.P2P.AddrBookStrict)

	pexReactor := pex.NewReactor(addrBook, &pex.ReactorConfig{
		SeedMode:                     true,
		Seeds:                        tmstrings.SplitAndTrim(cfg.P2P.BootstrapPeers, ",", " "),
		SeedDisconnectWaitPeriod:     15 * time.Minute, // default is 28 hours, we just want to harvest as many addresses as possible
		PersistentPeersMaxDialPeriod: 15 * time.Minute, // use exponential back-off
	})
	// TODO: CAN ask for addresses
	// pexReactor.ReceiveAddrs()

	transport := p2p.NewMConnTransport(
		noOpLogger, p2p.MConnConfig(cfg.P2P), []*p2p.ChannelDescriptor{},
		p2p.MConnTransportOptions{
			MaxAcceptedConnections: uint32(cfg.P2P.MaxNumInboundPeers),
		},
	)

	addr, err := types.NewNetAddressString(
		nodeKey.ID.AddressString(nodeInfo.ListenAddr),
	)
	if err != nil {
		logger.Error("Panic for chain " + cfg.PrettyName)
		panic(err)
	}
	if err := transport.Listen(p2p.NewEndpoint(addr)); err != nil {
		logger.Error("Panic for chain " + cfg.PrettyName)
		panic(err)
	}
	sw := p2p.NewSwitch(cfg.P2P, transport)

	sw.SetLogger(noOpLogger)
	sw.BaseService.SetLogger(noOpLogger)
	addrBook.SetLogger(noOpLogger)
	pexReactor.SetLogger(noOpLogger)

	sw.SetNodeKey(*nodeKey)
	sw.SetAddrBook(addrBook)
	sw.AddReactor("pex", pexReactor)

	// last
	sw.SetNodeInfo(nodeInfo)

	err = sw.Start()
	if err != nil {
		logger.Error("Panic for chain " + cfg.PrettyName)
		panic(err)
	}

	dialAddressBookPeers(addrBook, sw)
	tmos.TrapSignal(logger, func() {
		logger.Info("Shutting down chain " + cfg.PrettyName)
		addrBook.Save()
		_ = addrBook.Stop()
		_ = sw.Stop()
		_ = pexReactor.Stop()
	})

	return sw, addrBook
}

func dialAddressBookPeers(addrBook pex.AddrBook, sw *p2p.Switch) {
	addresses := addrBook.GetSelection() // this returns max 100 peers, but it's enough to start faster
	stringAddresses := make([]string, 0)
	for _, address := range addresses {
		stringAddresses = append(stringAddresses, address.String())
	}
	if len(stringAddresses) == 0 {
		return
	}
	logger.Info(fmt.Sprintf("Will dial %d random peers from existing address book", len(stringAddresses)))
	_ = sw.DialPeersAsync(stringAddresses)
}
