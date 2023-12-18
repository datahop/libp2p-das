package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	mrand "math/rand"

	dht "github.com/Blitz3r123/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	discovery "github.com/libp2p/go-libp2p-discovery"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/multiformats/go-multiaddr"
)

const builder_id = "12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73"

func NewHost(ctx context.Context, seed int64, port int, nodeType string) (host.Host, *dht.IpfsDHT, error) {
	var r io.Reader
	var crypto_code int

	if seed == 0 {
		r = rand.Reader
		crypto_code = crypto.RSA
	} else {
		r = mrand.New(mrand.NewSource(seed))
		crypto_code = crypto.Ed25519
	}

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto_code, 2048, r)
	if err != nil {
		return nil, nil, err
	}

	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))

	host, err := libp2p.New(ctx,
		libp2p.ListenAddrs(addr),
		libp2p.Identity(priv),
	)

	dht, err := NewDHT(ctx, host, nodeType)
	if err != nil {
		return nil, nil, err
	}

	// Connect the DHT instance to the host
	host.Peerstore().AddAddrs(dht.Host().ID(), dht.Host().Addrs(), peerstore.PermanentAddrTTL)
	routingDiscovery := discovery.NewRoutingDiscovery(dht)
	discovery.Advertise(ctx, routingDiscovery, "das")

	// Register an event handler for peer connection
	host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {

			if c.LocalPeer().String() == builder_id {
				node_suffix := ""
				if config.NodeType == "builder" {
					node_suffix = "B"
				} else if config.NodeType == "validator" {
					node_suffix = "V"
				} else {
					node_suffix = "R"
				}

				remote_peer_id := c.RemotePeer()

				routingTablePeerCountBefore := len(dht.RoutingTable().ListPeers())
				dht.RoutingTable().TryAddPeer(remote_peer_id, false, false)
				routingTablePeerCountAfter := len(dht.RoutingTable().ListPeers())

				if routingTablePeerCountBefore == routingTablePeerCountAfter {
					log.Printf("[%s - %s]: Failed to add peer %s to routing table\n", node_suffix, host.ID()[0:5].Pretty(), remote_peer_id[:5].Pretty())
				} else {
					log.Printf(
						"[%s - %s]: Peer %s connected to builder (%d -> %d connections)\n",
						node_suffix,
						host.ID()[0:5].Pretty(),
						remote_peer_id[:5].Pretty(),
						routingTablePeerCountBefore,
						routingTablePeerCountAfter,
					)
				}

			}
		},
	})

	return host, dht, nil
}
