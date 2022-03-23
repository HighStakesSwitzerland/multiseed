package main

import (
	"github.com/highstakesswitzerland/multiseed/internal/geoloc"
	"github.com/highstakesswitzerland/multiseed/internal/http"
	"github.com/highstakesswitzerland/multiseed/internal/seednode"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
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

	logger.Info("Starting Web Server...")
	http.StartWebServer(seedConfigs)

	seedSwitchs = seednode.StartSeedNodes(seedConfigs, nodeKey)

	StartGeolocServiceAndBlock(seedSwitchs)
}

func StartGeolocServiceAndBlock(seedSwitchs []p2p.Switch) {
	// Fire periodically
	for {
		select {
		case <-ticker.C:
			for _, sw := range seedSwitchs {
				peers := seednode.ToSeednodePeers(sw.Peers().List())
				geoloc.ResolveIps(peers, sw.NodeInfo().(p2p.DefaultNodeInfo).Network)
			}
		}
	}
}
