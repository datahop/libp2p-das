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
	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()

	// const TotalSamplesCount = 512 * 512
	const TotalSamplesCount = 10
	const TotalBlocksCount = 10

	blockID := 0

	// ? Generate 512 x 512 IDs for each sample
	sampleIDs := make([]int, TotalSamplesCount)

	for i := 0; i < TotalSamplesCount; i++ {
		sampleIDs[i] = i
	}

	var sample []byte = make([]byte, 512)

	currentBlockID := 0
	// ? x = blockID, y = sampleID
	var samplesReceived []int

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			if peerType == "builder" {

				if len(sampleIDs) == 0 && blockID < TotalBlocksCount {
					blockID += 1
					sampleIDs = make([]int, TotalSamplesCount)
					for i := 0; i < TotalSamplesCount; i++ {
						sampleIDs[i] = i
					}
				}

				// ? Get random sampleID
				sampleID := rand.Intn(TotalSamplesCount)
				// ? Remove the sampleID from the sampleIDs
				for i, id := range sampleIDs {
					if id == sampleID {
						sampleIDs = append(sampleIDs[:i], sampleIDs[i+1:]...)
						break
					}
				}
				// ? Decrease the length of the sampleIDs by 1
				if len(sampleIDs) > 0 {
					sampleIDs = sampleIDs[:len(sampleIDs)-1]
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
				putErr := dht.PutValue(ctx, "/das/sample/"+fmt.Sprint(blockID)+"/"+fmt.Sprint(sampleID), sample)

				if putErr != nil {
					log.Print("[BUILDER\t" + s.host.ID()[0:5].Pretty() + "] PutValue() Error: " + putErr.Error())
					log.Printf("[BUILDER\t"+s.host.ID()[0:5].Pretty()+"]: DHT Peers: %d\n", len(dht.RoutingTable().ListPeers()))
					stats.TotalFailedPuts += 1
					stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))
				} else {
					log.Print("[BUILDER\t" + s.host.ID()[0:5].Pretty() + "] " + colorize("PUT", "green") + " sample (" + fmt.Sprint(blockID) + ", " + fmt.Sprint(sampleID) + ") into DHT.\n")
					stats.TotalPutMessages += 1
					stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))
				}

			} else if peerType == "validator" {
				randomSampleID := rand.Intn(TotalSamplesCount)

				startTime := time.Now()
				_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(currentBlockID)+"/"+fmt.Sprint(randomSampleID))
				if err != nil {
					// log.Print("[VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] GetValue(/das/sample/" + fmt.Sprint(currentBlockID) + "/" + fmt.Sprint(randomSampleID) + ") Error: " + err.Error())
					stats.TotalFailedGets += 1
					stats.TotalGetMessages += 1
					stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
					stats.GetHops = append(stats.GetHops, hops)
				} else {
					log.Print("[VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] " + colorize("GET", "blue") + " sample (" + fmt.Sprint(blockID) + ", " + fmt.Sprint(randomSampleID) + ") from DHT.\n")
					stats.TotalGetMessages += 1
					stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
					stats.GetHops = append(stats.GetHops, hops)

					samplesReceived = append(samplesReceived, randomSampleID)
				}

				validatorRequiredSampleCount := 4*512 - 4 // ? 2 columns and 2 rows of samples (take away the 4 intersecting samples)

				if len(samplesReceived) == validatorRequiredSampleCount+75 { // ? 75 random samples
					currentBlockID += 1
					samplesReceived = make([]int, 0)
				}

			} else if peerType == "nonvalidator" {
				randomSampleID := rand.Intn(TotalSamplesCount)

				startTime := time.Now()

				_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(currentBlockID)+"/"+fmt.Sprint(randomSampleID))
				if err != nil {
					// log.Print("[NON VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] GetValue(/das/sample/" + fmt.Sprint(currentBlockID) + "/" + fmt.Sprint(randomSampleID) + ") Error: " + err.Error())
					stats.TotalFailedGets += 1
					stats.TotalGetMessages += 1
					stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
					stats.GetHops = append(stats.GetHops, hops)
				} else {
					log.Print("[NON VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] " + colorize("GET", "blue") + " sample (" + fmt.Sprint(blockID) + ", " + fmt.Sprint(randomSampleID) + ") from DHT.\n")
					stats.TotalGetMessages += 1
					stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
					stats.GetHops = append(stats.GetHops, hops)

					samplesReceived = append(samplesReceived, randomSampleID)
				}

				if len(samplesReceived) == 75 { // ? 75 random samples
					currentBlockID += 1
					samplesReceived = make([]int, 0)
				}

			}

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
