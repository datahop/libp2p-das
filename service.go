package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	dht "github.com/Blitz3r123/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	rpc "github.com/libp2p/go-libp2p-gorpc"
	// dht "./vendor/go-libp2p-kad-dht"
)

type Service struct {
	rpcServer *rpc.Server
	rpcClient *rpc.Client
	host      host.Host
	protocol  protocol.ID
}

func NewService(host host.Host, protocol protocol.ID) *Service {
	return &Service{
		host:     host,
		protocol: protocol,
	}
}

func (s *Service) SetupRPC() error {
	echoRPCAPI := EchoRPCAPI{service: s}

	s.rpcServer = rpc.NewServer(s.host, s.protocol)
	err := s.rpcServer.Register(&echoRPCAPI)
	if err != nil {
		return err
	}

	s.rpcClient = rpc.NewClientWithServer(s.host, s.protocol, s.rpcServer)
	return nil
}

func (s *Service) StartMessaging(dht *dht.IpfsDHT, stats *Stats, peerType string, ctx context.Context) {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	// ? Generate 2 IDs for the blocks
	blockIDs := make([]int, 2)

	// ? Generate 512 IDs for the columns
	colIDs := make([]int, 512)

	// ? Generate 512 IDs for the rows
	rowIDs := make([]int, 512)

	var sample []byte = make([]byte, 512)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			if peerType == "builder" {
				/*
					? The builder PUTS 512 x 512 512B samples where each sample key is a hash of the x, y, and block number i.e. hash the sample x, y, and block size and use that as key.
				*/

				// ? Generate random block ID
				blockID := rand.Intn(2)
				// ? Remove the colID from the colIDs
				for i, id := range blockIDs {
					if id == blockID {
						blockIDs = append(blockIDs[:i], blockIDs[i+1:]...)
						break
					}
				}
				// ? Decrease the length of the blockIDs by 1
				if len(blockIDs) > 0 {
					blockIDs = blockIDs[:len(blockIDs)-1]
				}

				// ? Get random colID
				colID := rand.Intn(512)
				// ? Remove the colID from the colIDs
				for i, id := range colIDs {
					if id == colID {
						colIDs = append(colIDs[:i], colIDs[i+1:]...)
						break
					}
				}
				// ? Decrease the length of the colIDs by 1
				if len(colIDs) > 0 {
					colIDs = colIDs[:len(colIDs)-1]
				}

				// ? Get random rowID
				rowID := rand.Intn(512)
				// ? Remove the rowID from the rowIDs
				for i, id := range rowIDs {
					if id == rowID {
						rowIDs = append(rowIDs[:i], rowIDs[i+1:]...)
						break
					}
				}
				// ? Decrease the length of the rowIDs by 1
				if len(rowIDs) > 0 {
					rowIDs = rowIDs[:len(rowIDs)-1]
				}

				peers := FilterSelf(s.host.Peerstore().Peers(), s.host.ID())
				dhtPeers := FilterSelf(dht.RoutingTable().ListPeers(), s.host.ID())

				if len(peers) == 0 && len(dhtPeers) == 0 {
					continue
				}

				if len(peers) == 0 || len(dhtPeers) == 0 {
					for _, p := range peers {
						_, err := dht.RoutingTable().TryAddPeer(p, false, true)
						if err != nil {
							log.Printf("Failed to add peer %s : %s\n", p[0:5].Pretty(), err.Error())
						}
					}
				}

				startTime := time.Now()

				// ? Put sample into DHT
				putErr := dht.PutValue(ctx, "/das/sample/"+s.host.ID().Pretty()+"/"+fmt.Sprint(blockID)+"/"+fmt.Sprint(colID)+"/"+fmt.Sprint(rowID), sample)

				if putErr != nil {
					log.Print("[BUILDER\t" + s.host.ID()[0:5].Pretty() + "] PutValue() Error: " + putErr.Error())
					log.Printf("[BUILDER\t"+s.host.ID()[0:5].Pretty()+"]: DHT Peers: %d\n", len(dht.RoutingTable().ListPeers()))
					stats.TotalFailedPuts += 1
					stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))
				}

				log.Print("[BUILDER\t" + s.host.ID()[0:5].Pretty() + "] " + colorize("PUT", "green") + " sample (" + fmt.Sprint(blockID) + ", " + fmt.Sprint(colID) + ", " + fmt.Sprint(rowID) + ") into DHT.\n")
				stats.TotalPutMessages += 1
				stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))

			} else if peerType == "validator" {
				continue
			}

			// if len(peers) > 0 && peerType == "validator" {
			// 	startTime := time.Now()
			// 	// ? Put sample into DHT
			// 	putErr := dht.PutValue(ctx, "/das/sample/"+s.host.ID().Pretty(), sample)
			// 	if putErr != nil {
			// 		stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))
			// 		log.Print("[" + s.host.ID()[0:5].Pretty() + "] PutValue() Error: " + putErr.Error())
			// 	}
			// 	log.Print("[" + s.host.ID()[0:5].Pretty() + "] " + colorize("PUT", "green") + " 42KB sample into DHT.\n")
			// 	stats.TotalPutMessages += 1
			// 	stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))
			// }

			// // ? Get random peer's sample from DHT
			// if len(peers) > 0 {
			// 	randIndex := rand.Intn(len(peers))
			// 	randomPeer := peers[randIndex]
			// 	startTime := time.Now()
			// 	// ? Get sample from DHT
			// 	_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+randomPeer.Pretty())
			// 	if err != nil {
			// 		// log.Print("[" + s.host.ID()[0:5].Pretty() + "] GetValue() Error: " + err.Error())
			// 		stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
			// 		stats.TotalFailedGets += 1
			// 		stats.TotalGetMessages += 1
			// 		stats.GetHops = append(stats.GetHops, hops)
			// 	} else {
			// 		log.Print("[" + s.host.ID()[0:5].Pretty() + "] " + colorize("GOT", "blue") + " 42KB sample for " + randomPeer[0:5].Pretty() + " from DHT.\n")
			// 		stats.TotalSuccessGets += 1
			// 		stats.TotalGetMessages += 1
			// 		stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
			// 		stats.GetHops = append(stats.GetHops, hops)
			// 	}
			// }

			// s.counter++
			// s.Echo(dht, samples, ctx)
		}
	}
}

func (s *Service) ReceiveEcho(envelope Envelope) Envelope {
	fmt.Printf("Peer %s got 42KB\n", s.host.ID())

	return Envelope{
		Message: fmt.Sprintf("Peer %s got 42KB", s.host.ID()),
	}
}

func FilterSelf(peers peer.IDSlice, self peer.ID) peer.IDSlice {
	var withoutSelf peer.IDSlice
	for _, p := range peers {
		if p != self {
			withoutSelf = append(withoutSelf, p)
		}
	}
	return withoutSelf
}

func Ctxts(n int) []context.Context {
	ctxs := make([]context.Context, n)
	for i := 0; i < n; i++ {
		ctxs[i] = context.Background()
	}
	return ctxs
}

func CopyEnvelopesToIfaces(in []*Envelope) []interface{} {
	ifaces := make([]interface{}, len(in))
	for i := range in {
		in[i] = &Envelope{}
		ifaces[i] = in[i]
	}
	return ifaces
}
