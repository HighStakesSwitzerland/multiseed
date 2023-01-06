package http

import (
	"embed"
	"encoding/json"
	"github.com/HighStakesSwitzerland/tendermint/libs/log"
	"github.com/highstakesswitzerland/multiseed/internal/geoloc"
	"github.com/highstakesswitzerland/multiseed/internal/seednode"
	"net/http"
)

var (
	logger = log.MustNewDefaultLogger("text", "info", false)
)

type WebResources struct {
	Res   embed.FS
	Files map[string]string
}

func StartWebServer(seedConfig *seednode.TSConfig) {
	// serve endpoint
	http.HandleFunc("/api/peers", writePeers)

	// start web server in non-blocking
	go func() {
		err := http.ListenAndServe(":"+seedConfig.HttpPort, nil)
		logger.Info("HTTP Server started", "port", seedConfig.HttpPort)
		if err != nil {
			panic(err)
		}
	}()
}

func writePeers(w http.ResponseWriter, r *http.Request) {
	marshal, err := json.Marshal(&geoloc.ResolvedPeers)
	if err != nil {
		logger.Info("Failed to marshal peers list")
		return
	}
	_, err = w.Write(marshal)
	if err != nil {
		return
	}
}
