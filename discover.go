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

				if h.Network().Connectedness(p.ID) != network.Connected {
					_, err = h.Network().DialPeer(ctx, p.ID)
					if err != nil {
						continue
					}
				}
			}

		}
	}
}
