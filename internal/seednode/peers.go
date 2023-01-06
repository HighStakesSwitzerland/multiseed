package seednode

import (
	"github.com/HighStakesSwitzerland/tendermint/internals/p2p"
	"github.com/HighStakesSwitzerland/tendermint/internals/p2p/pex"
	"github.com/HighStakesSwitzerland/tendermint/types"
)

var (
	peerList []*Peer
)

type Peer struct {
	Moniker string       `json:"moniker"`
	IP      string       `json:"-"` // IPs should not be sent to the frontend
	NodeId  types.NodeID `json:"-"`
}

/*
Returns the current reactor peers. As in seed mode the pex module disconnects quickly, this list can grow and shrink
according to the current connexions
*/
func ToSeednodePeers(peers []p2p.Peer) []*Peer {
	if len(peers) > 0 {
		//    logger.Info(fmt.Sprintf("Address book contains %d peers", len(peers)), "peers", peers)
		peerList = p2pPeersToPeerList(peers)
		return peerList
	}
	return nil
}

func p2pPeersToPeerList(list []p2p.Peer) []*Peer {
	var _peers []*Peer
	for _, p := range list {
		_peers = append(_peers, &Peer{
			Moniker: p.NodeInfo().Moniker,
			IP:      p.(pex.Peer).RemoteIP().String(),
			NodeId:  p.NodeInfo().ID(),
		})
	}
	return _peers
}
