package geoloc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/HighStakesSwitzerland/tendermint/internals/p2p"
	"github.com/HighStakesSwitzerland/tendermint/internals/p2p/pex"
	"github.com/HighStakesSwitzerland/tendermint/libs/log"
	"github.com/HighStakesSwitzerland/tendermint/types"
	"github.com/go-kit/kit/transport/http/jsonrpc"
	"github.com/highstakesswitzerland/multiseed/internal/seednode"
	"io"
	"io/ioutil"
	mrand "math/rand"
	"net"
	"net/http"
	"time"
)

var (
	ResolvedPeers = make(map[string]Chain)
	logger        = log.MustNewDefaultLogger("text", "info", false)
	ipApiUrl      = "http://ip-api.com/batch"
)

type Chain struct {
	ChainId    string              `json:"chain_id"`
	PrettyName string              `json:"pretty_name"`
	Nodes      []GeolocalizedPeers `json:"nodes"`
}

type GeolocalizedPeers struct {
	Moniker  string       `json:"moniker"`
	IP       net.IP       `json:"-"` // IPs should not be sent to the frontend
	Port     uint16       `json:"-"`
	NodeId   types.NodeID `json:"node_id"`
	LastSeen time.Time    `json:"last_seen"`
	Country  string       `json:"country"`
	Region   string       `json:"region"`
	City     string       `json:"city"`
	Lat      float32      `json:"lat"`
	Lon      float32      `json:"lon"`
	Isp      string       `json:"isp"`
	Org      string       `json:"org"`
	As       string       `json:"as"`
}

type ipServiceResponse struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region"`
	RegionName  string  `json:"region_name"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float32 `json:"lat"`
	Lon         float32 `json:"lon"`
	Timezone    string  `json:"timezone"`
	Isp         string  `json:"isp"`
	Org         string  `json:"org"`
	As          string  `json:"as"`
	Query       string  `json:"Query"`
}

/*
Resolve ips using https://ip-api.com/ geolocation free service
Appends the new resolved peers to the ResolvedPeers slice, so we keep the full list since the startup
*/
func ResolveIps(cfg seednode.SeedNodeConfig) {
	chainId := cfg.Sw.NodeInfo().Network
	chain := ResolvedPeers[chainId]
	geolocalizedPeers := resolve(get45UnresolvedPeers(cfg, chainId)) //will limit to 45 peers
	for _, peer := range geolocalizedPeers {
		// save the peer to the address book if it doesn't exist
		err := cfg.AddrBook.AddAddress(&p2p.NetAddress{
			ID:   peer.NodeId,
			IP:   peer.IP,
			Port: peer.Port,
		}, cfg.Sw.NetAddress())
		if err != nil {
			logger.Error("Error adding peer to address book: " + err.Error())
		}
		for _, address := range cfg.AddrBook.GetAddrbookContent() {
			if address.Addr.IP.String() == peer.IP.String() {
				// element exists in address book (hopefully always the case), we need to update it
				address.Org = peer.Org
				address.As = peer.As
				address.Isp = peer.Isp
				address.Lat = peer.Lat
				address.Lon = peer.Lon
				address.City = peer.City
				address.Region = peer.Region
				address.Country = peer.Country
				address.Moniker = peer.Moniker
			}
		}
	}
	for _, newPeer := range geolocalizedPeers {
		found := false
		for _, existingPeer := range chain.Nodes {
			if existingPeer.IP.Equal(newPeer.IP) {
				found = true
				existingPeer.LastSeen = newPeer.LastSeen
				existingPeer.NodeId = newPeer.NodeId
				existingPeer.Lat = newPeer.Lat
				existingPeer.Lon = newPeer.Lon
				existingPeer.Moniker = newPeer.Moniker
				existingPeer.Country = newPeer.Country
				existingPeer.Region = newPeer.Region
				existingPeer.City = newPeer.City
				existingPeer.Isp = newPeer.Isp
				existingPeer.As = newPeer.As
				existingPeer.Org = newPeer.Org
				break
			}
		}
		if !found {
			chain.Nodes = append(chain.Nodes, newPeer) // add new peer if not found
		}
	}
	ResolvedPeers[chainId] = chain
	logger.Info(fmt.Sprintf("We have %d total resolved peers for chain %s", len(ResolvedPeers[chainId].Nodes), cfg.Cfg.PrettyName))
}

func LoadSavedResolvedPeers(cfg seednode.SeedNodeConfig) {
	chain := ResolvedPeers[cfg.Cfg.ChainId]
	chain.ChainId = cfg.Cfg.ChainId
	chain.PrettyName = cfg.Cfg.PrettyName
	chain.Nodes = make([]GeolocalizedPeers, 0)

	for _, address := range cfg.AddrBook.GetAddrbookContent() {
		if address.Lat != 0 { // only add resolved nodes
			node := GeolocalizedPeers{
				Moniker:  address.Moniker,
				IP:       address.Addr.IP,
				Port:     address.Addr.Port,
				LastSeen: address.LastSuccess,
				Country:  address.Country,
				Region:   address.Region,
				City:     address.City,
				Lat:      address.Lat,
				Lon:      address.Lon,
				Isp:      address.Isp,
				Org:      address.Org,
				As:       address.As,
				NodeId:   address.ID(),
			}
			chain.Nodes = append(chain.Nodes, node)
		}
	}
	ResolvedPeers[cfg.Cfg.ChainId] = chain
}

func resolve(unresolvedPeers []*seednode.Peer) []GeolocalizedPeers {
	chunkSize := 10
	var geolocalizedPeers []GeolocalizedPeers
	peersLength := len(unresolvedPeers)

	for i := 0; i < peersLength; i += chunkSize {
		end := i + chunkSize
		if end > peersLength {
			end = peersLength
		}
		var chunk []*seednode.Peer
		chunk = append(chunk, unresolvedPeers[i:end]...)
		if len(chunk) > 0 {
			time.Sleep(1 * time.Second) // external service provider does not like fast queries...
			ipServiceResponses := fillGeolocData(chunk)
			var newGeolocalizedPeer GeolocalizedPeers
			if ipServiceResponses == nil {
				continue
			}
			for _, elt := range ipServiceResponses {
				if elt.Status != "success" {
					continue
				}
				peer := findPeerInList(elt, unresolvedPeers)
				if peer == nil {
					logger.Error("Could not find peer in existing list! It may have not been resolved by the service")
					continue
				}
				newGeolocalizedPeer = GeolocalizedPeers{
					Moniker:  peer.Moniker,
					LastSeen: peer.LastSeen,
					Country:  elt.Country,
					Region:   elt.Region,
					City:     elt.City,
					Lat:      elt.Lat,
					Lon:      elt.Lon,
					Isp:      elt.Isp,
					Org:      elt.Org,
					As:       elt.As,
					NodeId:   peer.NodeId,
					IP:       peer.IP,
					Port:     peer.Port,
				}
				geolocalizedPeers = append(geolocalizedPeers, newGeolocalizedPeer)
			}
		}
	}
	return geolocalizedPeers
}

func fillGeolocData(chunk []*seednode.Peer) []ipServiceResponse {
	logger.Info(fmt.Sprintf("Calling ip-api service with %d IPs", len(chunk)))
	var ipList []string

	for _, peer := range chunk {
		ipList = append(ipList, (*peer).IP.String())
	}

	payload, err := json.Marshal(ipList)
	if err != nil {
		logger.Error("Failed to marshal peers list for geoloc service", err)
		return nil
	}

	post, err := http.Post(ipApiUrl, jsonrpc.ContentType, bytes.NewBuffer(payload))
	if err != nil {
		logger.Error(fmt.Sprintf("IP geoloc service returned an error: %s", err.Error()))
		return nil
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("Error while waiting for response", err)
		}
	}(post.Body)

	body, err := ioutil.ReadAll(post.Body)
	if err != nil {
		logger.Error("Error reading reponse", err)
	}

	//Decode the data
	response := make([]ipServiceResponse, 0)
	if err := json.Unmarshal(body, &response); err != nil {
		logger.Error("Error while unmarshalling response", err)
	}

	return response
}

// We limit to 45 peers because of the rate limit of ip-api external service (45 per minute)
func get45UnresolvedPeers(cfg seednode.SeedNodeConfig, chain string) []*seednode.Peer {
	var peersToResolve []*seednode.Peer

	for _, peer := range seednode.ToSeednodePeers(cfg.Sw.Peers().List()) {
		if !isResolved(*peer, chain) {
			peersToResolve = append(peersToResolve, peer)
		}
		if len(peersToResolve) == 45 {
			break
		}
	}
	if len(peersToResolve) < 45 {
		// fill with unresolved peers from addressbook
		// TODO: also get older peers that could be refreshed? Or remove them definitively?
		knownAddresses := getRandomPeersFromAddrBook(cfg.AddrBook.GetAddrbookContent())
		for _, address := range knownAddresses {
			if len(address.Country) == 0 {
				peer := &seednode.Peer{
					Moniker:  address.Moniker,
					IP:       address.Addr.IP,
					NodeId:   address.ID(),
					LastSeen: address.LastSuccess,
				}
				peersToResolve = append(peersToResolve, peer)
			}
			if len(peersToResolve) == 45 {
				break
			}
		}
	}
	logger.Info(fmt.Sprintf("Exit from get45UnresolvedPeers with %d peers", len(peersToResolve)))
	return peersToResolve
}

func isResolved(peer seednode.Peer, chain string) bool {
	for _, elt := range ResolvedPeers[chain].Nodes {
		if elt.IP.String() == peer.IP.String() {
			return true
		}
	}
	return false
}

func findPeerInList(ipServiceResponse ipServiceResponse, peer []*seednode.Peer) *seednode.Peer {
	for _, elt := range peer {
		if (*elt).IP.String() == ipServiceResponse.Query { // TODO: what on ipv6
			return elt
		}
	}
	return nil
}

func getRandomPeersFromAddrBook(addrbook []*pex.KnownAddress) []*pex.KnownAddress {
	// XXX: instead of making a list of all addresses, shuffling, and slicing a random chunk,
	// could we just select a random numAddresses of indexes?
	allAddr := make([]*pex.KnownAddress, 0)
	for _, ka := range addrbook {
		if ka.LastSuccess.Year() == 1 {
			continue // ignore peers we have never connected to
		}
		allAddr = append(allAddr, ka)
	}

	// Fisher-Yates shuffle the array. We only need to do the first
	// `numAddresses' since we are throwing the rest.
	len := len(allAddr)
	for i := 0; i < len; i++ {
		// pick a number between current index and the end
		// nolint:gosec // G404: Use of weak random number generator
		j := mrand.Intn(len-i) + i
		allAddr[i], allAddr[j] = allAddr[j], allAddr[i]
	}

	// slice off the limit we are willing to share.
	max := len
	if len > 45 {
		max = 45
	}
	return allAddr[:max]
}
