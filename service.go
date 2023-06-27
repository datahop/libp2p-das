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
	counter   int
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

func (s *Service) StartMessaging(dht *dht.IpfsDHT, stats *Stats, isValidator bool, ctx context.Context) {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var sample []byte = make([]byte, 42000)
			peers := FilterSelf(s.host.Peerstore().Peers(), s.host.ID())

			if len(peers) > 0 && isValidator {
				startTime := time.Now()
				// ? Put sample into DHT
				putErr := dht.PutValue(ctx, "/das/sample/"+s.host.ID().Pretty(), sample)
				if putErr != nil {
					stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))
					log.Print("[" + s.host.ID()[0:5].Pretty() + "] PutValue() Error: " + putErr.Error())
				}
				log.Print("[" + s.host.ID()[0:5].Pretty() + "] " + colorize("PUT", "green") + " 42KB sample into DHT.\n")
				stats.TotalPutMessages += 1
				stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))
			}

			// ? Get random peer's sample from DHT
			if len(peers) > 0 {
				randIndex := rand.Intn(len(peers))
				randomPeer := peers[randIndex]
				startTime := time.Now()
				// ? Get sample from DHT
				_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+randomPeer.Pretty())
				if err != nil {
					log.Print("[" + s.host.ID()[0:5].Pretty() + "] GetValue() Error: " + err.Error())
					stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
					stats.TotalFailedGets += 1
					stats.TotalGetMessages += 1
					stats.GetHops = append(stats.GetHops, hops)
				} else {
					log.Print("[" + s.host.ID()[0:5].Pretty() + "] " + colorize("GOT", "blue") + " 42KB sample for " + randomPeer[0:5].Pretty() + " from DHT.\n")
					stats.TotalSuccessGets += 1
					stats.TotalGetMessages += 1
					stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
					stats.GetHops = append(stats.GetHops, hops)
				}
			}

			// s.counter++
			// s.Echo(dht, samples, ctx)
		}
	}
}

func (s *Service) Echo(dht *dht.IpfsDHT, samples []byte, ctx context.Context) {

	// sampleKey := "sample key"
	// if value, err := dht.GetValue(ctx, sampleKey); err != nil {
	//  fmt.Println("Error getting value from DHT:")
	//  log.Fatal(err)
	// } else if value != nil {
	//  fmt.Println("Got value from DHT: ", value)
	// }

	// peers := FilterSelf(s.host.Peerstore().Peers(), s.host.ID())
	// var replies = make([]*Envelope, len(peers))

	// errs := s.rpcClient.MultiCall(
	//  Ctxts(len(peers)),
	//  peers,
	//  EchoService,
	//  EchoServiceFuncEcho,
	//  Envelope{Samples: samples},
	//  CopyEnvelopesToIfaces(replies),
	// )

	// for i, err := range errs {
	//  if err != nil {
	//      fmt.Printf("Peer %s returned error: %-v\n", peers[i].Pretty(), err)
	//  }
	// }
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
