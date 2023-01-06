package seednode

import (
	"github.com/HighStakesSwitzerland/tendermint/internals/p2p"
	"github.com/HighStakesSwitzerland/tendermint/types"
	"net"
	"time"
)

type Peer struct {
	Moniker  string
	IP       net.IP
	Port     uint16
	NodeId   types.NodeID
	LastSeen time.Time
}

func ToSeednodePeers(peers []p2p.Peer) []*Peer {
	if len(peers) > 0 {
		return p2pPeersToPeerList(peers)
	}
	return nil
}

func p2pPeersToPeerList(list []p2p.Peer) []*Peer {
	var _peers []*Peer
	for _, p := range list {
		_peers = append(_peers, &Peer{
			Moniker:  p.NodeInfo().Moniker,
			LastSeen: time.Now().Add(-p.Status().Duration), //TODO: unsure this is accurate
			IP:       p.SocketAddr().IP,
			Port:     p.SocketAddr().Port,
			NodeId:   p.NodeInfo().ID(),
		})
	}
	return _peers
}
