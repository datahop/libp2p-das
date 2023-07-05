package main

import (
	"context"
	"log"
	"time"

	dht "github.com/Blitz3r123/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	discovery "github.com/libp2p/go-libp2p-discovery"
)

func Discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, rendezvous string) {
	var routingDiscovery = discovery.NewRoutingDiscovery(dht)

	discovery.Advertise(ctx, routingDiscovery, rendezvous)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			peers, err := discovery.FindPeers(ctx, routingDiscovery, rendezvous)
			if err != nil {
				log.Printf("FindPeers failed\n")
				log.Fatal(err)
			}

			for _, p := range peers {
				if p.ID == h.ID() {
					continue
				}

				// ? If the peer is not already on the network, dial it
				if h.Network().Connectedness(p.ID) != network.Connected {
					_, err = h.Network().DialPeer(ctx, p.ID)
					if err != nil {
						continue
					}

					// ? Try to add the peer to the dht routing table
					_, err := dht.RoutingTable().TryAddPeer(p.ID, true, true)
					if err != nil {
						log.Printf("Failed to add peer to dht routing table\n")
						log.Fatal(err)
					}

				} else {
					// ? Check if the peer is already in the dht routing table
					_, err := dht.RoutingTable().TryAddPeer(p.ID, true, true)
					if err != nil {
						log.Printf("Failed to add peer to dht routing table\n")
						log.Fatal(err)
					}
				}
			}

			// ? Check if the number of peers and number of dht routing table peers are the same
			if (len(peers) > 0) && (len(peers)-1 != len(dht.RoutingTable().ListPeers())) {
				log.Printf("Found %d peers and %d dht peers\n", len(peers), len(dht.RoutingTable().ListPeers()))

			}

		}
	}
}
