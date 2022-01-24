package seednode

import (
  "fmt"
  "github.com/mitchellh/go-homedir"
  "github.com/tendermint/tendermint/libs/log"
  tmstrings "github.com/tendermint/tendermint/libs/strings"
  "github.com/tendermint/tendermint/p2p"
  "github.com/tendermint/tendermint/p2p/pex"
  "github.com/tendermint/tendermint/version"
  "os"
  "path/filepath"
  "time"
)

var (
  logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "config")
)

func StartSeedNodes(seedConfig TSConfig, nodeKey p2p.NodeKey) []p2p.Switch {
  var switches []p2p.Switch
  if sw := startSeedNode(seedConfig.Terra, nodeKey); sw != nil {
    switches = append(switches, *sw)
  }
  if sw := startSeedNode(seedConfig.Band, nodeKey); sw != nil {
    switches = append(switches, *sw)
  }

  return switches
}

func startSeedNode(config P2PConfig, nodeKey p2p.NodeKey) *p2p.Switch {
  if config.Enable == false {
    return nil
  }

  logger.Info("Starting Seed Node for chain " + config.ChainId)

  protocolVersion :=
    p2p.NewProtocolVersion(
      version.P2PProtocol,
      version.BlockProtocol,
      0,
    )

  // NodeInfo gets info on your node
  nodeInfo := p2p.DefaultNodeInfo{
    ProtocolVersion: protocolVersion,
    DefaultNodeID:   nodeKey.ID(),
    ListenAddr:      config.ListenAddress,
    Network:         config.ChainId,
    Version:         "1.0.0",
    Channels:        []byte{pex.PexChannel},
    Moniker:         fmt.Sprintf("%s-multiseed", config.ChainId),
  }

  addr, err := p2p.NewNetAddressString(p2p.IDAddressString(nodeInfo.DefaultNodeID, nodeInfo.ListenAddr))
  if err != nil {
    panic(err)
  }

  transport := p2p.NewMultiplexTransport(nodeInfo, nodeKey, p2p.MConnConfig(&config.P2PConfig))
  if err := transport.Listen(*addr); err != nil {
    panic(err)
  }

  userHomeDir, _ := homedir.Dir()
  addrBookFilePath := filepath.Join(userHomeDir, ".multiseed", "addrbook.json")
  addrBook := pex.NewAddrBook(addrBookFilePath, config.AddrBookStrict)
  addrBook.SetLogger(logger.With("module", "addrbook", "chain", config.ChainId))

  pexReactor := pex.NewReactor(addrBook, &pex.ReactorConfig{
    SeedMode:                     true,
    Seeds:                        tmstrings.SplitAndTrim(config.Seeds, ",", " "),
    SeedDisconnectWaitPeriod:     1 * time.Second, // default is 28 hours, we just want to harvest as many addresses as possible
    PersistentPeersMaxDialPeriod: 5 * time.Minute, // use exponential back-off
  })

  sw := p2p.NewSwitch(&config.P2PConfig, transport)
  sw.SetNodeKey(&nodeKey)
  sw.SetAddrBook(addrBook)
  sw.AddReactor("pex", pexReactor)

  //sw.SetLogger(logger.With("module", "switch"))
  //pexReactor.SetLogger(logger.With("module", "pex"))

  // last
  sw.SetNodeInfo(nodeInfo)

  err = sw.Start()
  if err != nil {
    panic(err)
  }

  return sw
}
