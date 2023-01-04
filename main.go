package main

import (
	"github.com/HighStakesSwitzerland/tendermint/internals/p2p"
	"github.com/HighStakesSwitzerland/tendermint/libs/log"
	"github.com/highstakesswitzerland/multiseed/internal/geoloc"
	"github.com/highstakesswitzerland/multiseed/internal/http"
	"github.com/highstakesswitzerland/multiseed/internal/seednode"
	"time"
)

var (
	logger = log.MustNewDefaultLogger("text", "info", false)
	ticker = time.NewTicker(60 * time.Second)
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
