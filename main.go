package main

import (
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/p2p"
	"github.com/terran-stakers/multiseed/internal/geoloc"
	"github.com/terran-stakers/multiseed/internal/seednode"
	"os"
	"time"
)

var (
	logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "main")
	ticker = time.NewTicker(5 * time.Second)
)

func main() {
	seedConfigs, nodeKey := seednode.InitConfigs()
	var seedSwitchs []p2p.Switch

	//	logger.Info("Starting Web Server...")
	//	http.StartWebServer(seedConfigs, geolocalizedIps)

	seedSwitchs = seednode.StartSeedNodes(seedConfigs, nodeKey)

	tmos.TrapSignal(logger, func() {
		logger.Info("shutting down...")
		ticker.Stop()
		for _, sw := range seedSwitchs {
			_ = sw.Stop()
		}
	})

	StartGeolocServiceAndBlock(seedSwitchs)
}

func StartGeolocServiceAndBlock(seedSwitchs []p2p.Switch) {
	// Fire periodically
	for {
		select {
		case <-ticker.C:
			var peersFromAllNodes []p2p.Peer
			for _, sw := range seedSwitchs {
				peersFromAllNodes = append(peersFromAllNodes, sw.Peers().List()...)
			}
			peers := seednode.GetPeers(peersFromAllNodes)
			geoloc.ResolveIps(peers)
		}
	}
}
