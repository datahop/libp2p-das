package main

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	dht "github.com/Blitz3r123/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	discovery "github.com/libp2p/go-libp2p-discovery"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/multiformats/go-multiaddr"
)

type Config struct {
	NodeType       string
	ParcelSize     int
	Port           int
	ProtocolID     string
	Rendezvous     string
	Seed           int64
	DiscoveryPeers addrList
	Debug          bool
}

type Stats struct {
	// Operations
	BlockIDs        []string
	ParcelKeyHashes []string
	ParcelStatuses  []string
	PutTimestamps   []time.Time
	PutLatencies    []time.Duration
	GetTimestamps   []time.Time
	GetLatencies    []time.Duration
	GetHops         []int

	// Total Stats
	TotalPutMessages int
	TotalFailedPuts  int
	TotalSuccessPuts int
	TotalGetMessages int
	TotalFailedGets  int
	TotalSuccessGets int

	// Latencies
	SeedingLatencies        []time.Duration
	RowSamplingLatencies    []time.Duration
	ColSamplingLatencies    []time.Duration
	RandomSamplingLatencies []time.Duration
	TotalSamplingLatencies  []time.Duration
}

var config Config

func main() {
	// Turn on/off logging messages in stdout
	// log.SetOutput(ioutil.Discard)
	log.SetOutput(os.Stdout)

	stats := &Stats{}

	// flag.StringVar(&config.Rendezvous, "rendezvous", "/das", "")
	flag.StringVar(&config.NodeType, "nodeType", "validator", "The node type to run (validator, nonvalidator, builder)")
	flag.IntVar(&config.ParcelSize, "parcelSize", 512, "The size of the parcels to send - make sure 512 divides evenly by this number")
	flag.Int64Var(&config.Seed, "seed", 0, "Seed value for generating a PeerID, 0 is random")
	flag.Var(&config.DiscoveryPeers, "peer", "Peer multiaddress for peer discovery")
	flag.StringVar(&config.ProtocolID, "protocolid", "/p2p/rpc", "")
	flag.IntVar(&config.Port, "port", 0, "")
	flag.Parse()

	nodeType := strings.ToLower(config.NodeType)
	nodeTypeSuffix := ""

	if nodeType == "builder" {
		nodeTypeSuffix = "B"
	} else if nodeType == "validator" {
		nodeTypeSuffix = "V"
	} else {
		nodeTypeSuffix = "R"
	}

	// h, dht, err := NewHost(context.Background(), config.Seed, config.Port, nodeType)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	var r io.Reader
	var crypto_code int

	if config.Seed == 0 {
		r = rand.Reader
		crypto_code = crypto.RSA
	} else {
		r = mrand.New(mrand.NewSource(config.Seed))
		crypto_code = crypto.Ed25519
	}

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto_code, 2048, r)
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", config.Port))

	h, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(addr),
		libp2p.Identity(priv),
	)
	if err != nil {
		log.Fatal(err)
	}

	dht, err := NewDHT(context.Background(), h, nodeType)
	if err != nil {
		log.Fatal(err)
	}

	h.Peerstore().AddAddrs(dht.Host().ID(), dht.Host().Addrs(), peerstore.PermanentAddrTTL)
	routingDiscovery := discovery.NewRoutingDiscovery(dht)
	discovery.Advertise(context.Background(), routingDiscovery, "das")

	// Register an event handler for peer connection
	h.Network().Notify(&network.NotifyBundle{
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
					log.Printf("[%s - %s]: Failed to add peer %s to routing table\n", node_suffix, h.ID()[0:5].Pretty(), remote_peer_id[:5].Pretty())
				} else {
					log.Printf(
						"[%s - %s]: Peer %s connected to builder (%d -> %d connections)\n",
						node_suffix,
						h.ID()[0:5].Pretty(),
						remote_peer_id[:5].Pretty(),
						routingTablePeerCountBefore,
						routingTablePeerCountAfter,
					)
				}

			}
		},
	})

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	if nodeType == "builder" {

		log.Printf("[B - %s] Builder started: %s\n", h.ID().Pretty()[:5], h.ID().Pretty())

	} else {

		wg.Add(1)
		go waitForBuilder(&wg, config.DiscoveryPeers, h, dht)
		wg.Wait()

		log.Printf("[%s - %s] Peer started: %s\n", nodeTypeSuffix, h.ID().Pretty()[:5], h.ID().Pretty()[:5])

	}

	service := NewService(h, protocol.ID(config.ProtocolID))
	err = service.SetupRPC()
	if err != nil {
		log.Fatal(err)
	}

	service.StartMessaging(h, dht, stats, nodeType, config.ParcelSize, ctx)

	if filename, err := writeOperationsToFile(stats, h, nodeType); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("[%s - %s] Operations written to %s\n", nodeTypeSuffix, h.ID()[0:5].Pretty(), filename)
	}

	if filename, err := writeTotalStatsToFile(stats, h, nodeType); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("[%s - %s] Total Stats written to %s\n", nodeTypeSuffix, h.ID()[0:5].Pretty(), filename)
	}

	if filename, err := writeLatencyStatsToFile(stats, h, nodeType); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("[%s - %s] Latencies written to %s\n", nodeTypeSuffix, h.ID()[0:5].Pretty(), filename)
	}

	cancel()

}

func writeTotalStatsToFile(stats *Stats, h host.Host, nodeType string) (string, error) {
	filename := h.ID()[0:10].Pretty() + "_total_stats_" + nodeType + ".csv"

	f, err := os.Create(filename)
	if err != nil {
		return filename, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	headers := []string{"Total PUT messages", "Total failed PUTs", "Total successful PUTs", "Total GET messages", "Total failed GETs", "Total successful GETs"}

	rows := [][]string{
		{strconv.Itoa(stats.TotalPutMessages), strconv.Itoa(stats.TotalFailedPuts), strconv.Itoa(stats.TotalSuccessPuts), strconv.Itoa(stats.TotalGetMessages), strconv.Itoa(stats.TotalFailedGets), strconv.Itoa(stats.TotalSuccessGets)},
	}

	// Write headers and rows to CSV file
	w.Write(headers)
	w.WriteAll(rows)
	if err := w.Error(); err != nil {
		return filename, err
	}

	return filename, nil
}

func writeOperationsToFile(stats *Stats, h host.Host, nodeType string) (string, error) {
	filename := h.ID()[0:10].Pretty() + "_operations_" + nodeType + ".csv"

	// Convert latencies and hops to rows
	var operationRows [][]string
	for i := 0; i < len(stats.BlockIDs) || i < len(stats.ParcelKeyHashes) || i < len(stats.ParcelStatuses) || i < len(stats.PutTimestamps) || i < len(stats.GetTimestamps) || i < len(stats.GetHops) || i < len(stats.PutLatencies) || i < len(stats.GetLatencies); i++ {
		var row []string

		if i < len(stats.BlockIDs) {
			row = append(row, stats.BlockIDs[i])
		} else {
			row = append(row, "")
		}

		if i < len(stats.ParcelKeyHashes) {
			row = append(row, stats.ParcelKeyHashes[i])
		} else {
			row = append(row, "")
		}

		if i < len(stats.ParcelStatuses) {
			row = append(row, stats.ParcelStatuses[i])
		} else {
			row = append(row, "")
		}

		if i < len(stats.PutTimestamps) {
			row = append(row, stats.PutTimestamps[i].String())
		} else {
			row = append(row, "")
		}

		if i < len(stats.PutLatencies) {
			row = append(row, strconv.FormatInt(stats.PutLatencies[i].Microseconds(), 10))
		} else {
			row = append(row, "")
		}

		if i < len(stats.GetTimestamps) {
			row = append(row, stats.GetTimestamps[i].String())
		} else {
			row = append(row, "")
		}

		if i < len(stats.GetLatencies) {
			row = append(row, strconv.FormatInt(stats.GetLatencies[i].Microseconds(), 10))
		} else {
			row = append(row, "")
		}

		if i < len(stats.GetHops) {
			row = append(row, strconv.Itoa(stats.GetHops[i]))
		} else {
			row = append(row, "")
		}

		operationRows = append(operationRows, row)
	}

	// Write latency stats to CSV file
	f, err := os.Create(filename)
	if err != nil {
		return filename, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	headers := []string{"Block ID", "Parcel Key Hashes", "Parcel Status", "PUT timestamps", "PUT latencies", "GET timestamps", "GET latencies", "GET hops"}
	rows := operationRows

	// Write headers and rows to CSV file
	w.Write(headers)
	w.WriteAll(rows)
	if err := w.Error(); err != nil {
		return filename, err
	}

	return filename, nil
}

func writeLatencyStatsToFile(stats *Stats, h host.Host, nodeType string) (string, error) {
	filename := h.ID()[0:10].Pretty() + "_latency_stats_" + nodeType + ".csv"

	// Convert latencies and hops to rows
	var latencyRows [][]string

	for i := 0; i < len(stats.SeedingLatencies) || i < len(stats.RowSamplingLatencies) || i < len(stats.ColSamplingLatencies) || i < len(stats.RandomSamplingLatencies) || i < len(stats.TotalSamplingLatencies); i++ {
		var row []string

		if i < len(stats.SeedingLatencies) {
			row = append(row, strconv.FormatInt(stats.SeedingLatencies[i].Microseconds(), 10))
		} else {
			row = append(row, "")
		}

		if i < len(stats.RowSamplingLatencies) {
			row = append(row, strconv.FormatInt(stats.RowSamplingLatencies[i].Microseconds(), 10))
		} else {
			row = append(row, "")
		}

		if i < len(stats.ColSamplingLatencies) {
			row = append(row, strconv.FormatInt(stats.ColSamplingLatencies[i].Microseconds(), 10))
		} else {
			row = append(row, "")
		}

		if i < len(stats.RandomSamplingLatencies) {
			row = append(row, strconv.FormatInt(stats.RandomSamplingLatencies[i].Microseconds(), 10))
		} else {
			row = append(row, "")
		}

		if i < len(stats.TotalSamplingLatencies) {
			row = append(row, strconv.FormatInt(stats.TotalSamplingLatencies[i].Microseconds(), 10))
		} else {
			row = append(row, "")
		}

		latencyRows = append(latencyRows, row)
	}

	// Write latency stats to CSV file
	f, err := os.Create(filename)
	if err != nil {
		return filename, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	headers := []string{"Seeding Latency (us)", "Row Sampling Latency (us)", "Col Sampling Latency (us)", "Random Sampling Latency (us)", "Total Sampling Latency (us)"}
	rows := latencyRows

	// Write headers and rows to CSV file
	w.Write(headers)
	w.WriteAll(rows)
	if err := w.Error(); err != nil {
		return filename, err
	}

	return filename, nil
}

type addrList []multiaddr.Multiaddr

func (al *addrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (al *addrList) Set(value string) error {
	addr, err := multiaddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*al = append(*al, addr)
	return nil
}

func waitForBuilder(wg *sync.WaitGroup, discoveryPeers addrList, h host.Host, dht *dht.IpfsDHT) {
	defer wg.Done()

	// ? Wait for a couple of seconds to make sure bootstrap peer is up and running
	time.Sleep(2 * time.Second)

	// ? Timeout of 10 seconds to connect to bootstrap peer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ? Connect to bootstrap peers
	for _, peerAddr := range discoveryPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)

		if err := h.Connect(ctx, *peerinfo); err != nil {

			log.Print()
			log.Printf("Error connecting to bootstrap node %q: %-v", peerinfo, err)
			log.Printf("peerinfo: %s\n", peerinfo)
			log.Printf("peerinfo.ID: %s\n", peerinfo.ID)
			log.Printf("peerinfo.Addrs: %s\n", peerinfo.Addrs)
			log.Printf("err: %s\n", err)
			log.Print()

		} else {

			if _, err := dht.FindPeer(ctx, peerinfo.ID); err != nil {

				log.Printf("Error finding peer: %s\n", err)

			} else {

				return
			}

		}
	}

	log.Printf("Could not connect to any bootstrap nodes...")

}

// func onPeerConnected(h host.Host, remote_peer_id peer.ID, dht *dht.IpfsDHT) (returningDht *dht.IpfsDHT) {

// 	node_suffix := ""
// 	if config.NodeType == "builder" {
// 		node_suffix = "B"
// 	} else if config.NodeType == "validator" {
// 		node_suffix = "V"
// 	} else {
// 		node_suffix = "R"
// 	}

// 	routingTablePeerCountBefore := len(dht.RoutingTable().ListPeers())
// 	dht.RoutingTable().TryAddPeer(remote_peer_id, false, false)
// 	routingTablePeerCountAfter := len(dht.RoutingTable().ListPeers())

// 	if routingTablePeerCountBefore == routingTablePeerCountAfter {
// 		log.Printf("[%s - %s]: Failed to add peer %s to routing table\n", node_suffix, h.ID()[0:5].Pretty(), remote_peer_id[:5].Pretty())
// 		return
// 	}

// 	log.Printf(
// 		"[%s - %s]: Peer %s connected to builder (%d -> %d connections)\n",
// 		node_suffix,
// 		h.ID()[0:5].Pretty(),
// 		remote_peer_id[:5].Pretty(),
// 		routingTablePeerCountBefore,
// 		routingTablePeerCountAfter,
// 	)

// 	return dht

// }
