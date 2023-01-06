package main

import (
	"github.com/HighStakesSwitzerland/tendermint/libs/log"
	"github.com/highstakesswitzerland/multiseed/internal/geoloc"
	"github.com/highstakesswitzerland/multiseed/internal/http"
	"github.com/highstakesswitzerland/multiseed/internal/seednode"
	"time"
)

var (
	logger = log.MustNewDefaultLogger("text", "info", false)
	ticker = time.NewTicker(60 * time.Second) // should stay 60 sec to match the ip-api service rate limit
)

func main() {
	seedConfigs, nodeKey := seednode.InitConfigs()
	var seedSwitchs []seednode.SeedNodeConfig

	logger.Info("Starting Web Server on port " + seedConfigs.HttpPort)
	http.StartWebServer(seedConfigs)

	seedSwitchs = seednode.StartSeedNodes(seedConfigs, &nodeKey)

	for _, config := range seedSwitchs {
		geoloc.LoadSavedResolvedPeers(config)
	}
	StartGeolocServiceAndBlock(seedSwitchs)
}

func StartGeolocServiceAndBlock(seedNodes []seednode.SeedNodeConfig) {
	// Fire periodically
	for {
		select {
		case <-ticker.C:
			for _, seedNodeConfig := range seedNodes {
				geoloc.ResolveIps(seedNodeConfig)
			}
		}
	}
}
