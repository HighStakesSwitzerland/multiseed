package main

import (
	"github.com/HighStakesSwitzerland/tendermint/libs/log"
	"github.com/highstakesswitzerland/multiseed/internal/config"
	"github.com/highstakesswitzerland/multiseed/internal/geoloc"
	"github.com/highstakesswitzerland/multiseed/internal/http"
	"github.com/highstakesswitzerland/multiseed/internal/seednode"
	"time"
)

var (
	logger = log.MustNewDefaultLogger("text", "info", false)
	ticker = time.NewTicker(300 * time.Second) // should staymin 60 sec to match the ip-api service rate limit
)

func main() {
	seedConfigs, nodeKey := config.InitConfigs()
	var seedSwitchs []seednode.SeedNodeConfig

	logger.Info("Starting Web Server on port " + seedConfigs.HttpPort)
	http.StartWebServer(seedConfigs)

	seedSwitchs = seednode.StartSeedNodes(seedConfigs, &nodeKey)

	for _, cfg := range seedSwitchs {
		geoloc.LoadSavedResolvedPeers(cfg)
	}
	StartGeolocServiceAndBlock(seedSwitchs)
}

func StartGeolocServiceAndBlock(seedNodes []seednode.SeedNodeConfig) {
	// Fire periodically
	for {
		select {
		case <-ticker.C:
			for _, seedNodeConfig := range seedNodes {
				seednode.SaveLastSeenAttrInAddrbook(seedNodeConfig) // update LastSeen values in address book at it is not done automatically on seed mode reactor
				geoloc.ResolveIps(seedNodeConfig)
			}
		}
	}
}
