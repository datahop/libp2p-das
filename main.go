package main

import (
	"context"
	"encoding/csv"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
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
	TotalPutMessages int
	TotalFailedPuts  int
	TotalGetMessages int
	TotalFailedGets  int
	TotalSuccessGets int

	// ? How long it took the builder to put a block into DHT
	SeedingLatencies []time.Duration

	// ? Array of latencies for puts
	PutLatencies []time.Duration
	// ? Array of latencies for gets
	GetLatencies []time.Duration
	// ? How long it takes to get 2 rows
	RowSamplingLatencies []time.Duration
	// ? How long it takes to get 2 cols
	ColSamplingLatencies []time.Duration
	// ? How long it takes to get 75 random cells
	RandomSamplingLatencies []time.Duration
	// ? Array of hops for gets
	GetHops []int
}

// func colorize(word string, colorName string) string {
// 	var c *color.Color
// 	switch colorName {
// 	case "red":
// 		c = color.New(color.FgRed)
// 	case "green":
// 		c = color.New(color.FgGreen)
// 	case "yellow":
// 		c = color.New(color.FgYellow)
// 	case "blue":
// 		c = color.New(color.FgBlue)
// 	default:
// 		c = color.New(color.Reset)
// 	}

// 	return c.Sprint(word)
// }

func main() {
	config := Config{}
	stats := &Stats{}

	// flag.StringVar(&config.Rendezvous, "rendezvous", "/das", "")
	flag.StringVar(&config.NodeType, "nodeType", "validator", "The node type to run (validator, nonvalidator, builder)")
	flag.IntVar(&config.ParcelSize, "parcelSize", 512, "The size of the parcels to send - make sure 512 divides evenly into this number")
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

	h, err := NewHost(context.Background(), config.Seed, config.Port)
	if err != nil {
		log.Fatal(err)
	}

	dht, err := NewDHT(context.Background(), h, nodeType)
	if err != nil {
		log.Printf("Error creating dht\n")
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	if nodeType == "builder" {
		wg.Add(1)
		go startBuilder(&wg, ctx, config.Seed, config.Port)

		service := NewService(h, protocol.ID(config.ProtocolID))
		err = service.SetupRPC()
		if err != nil {
			log.Fatal(err)
		}

		service.StartMessaging(dht, stats, nodeType, config.ParcelSize, ctx)

	} else {
		log.Printf("[%s - %s] Peer ID: %s\n", nodeTypeSuffix, h.ID().Pretty()[:5], h.ID().Pretty())

		wg.Add(1)
		go waitForBuilder(&wg, config.DiscoveryPeers, h)

		service := NewService(h, protocol.ID(config.ProtocolID))
		err = service.SetupRPC()
		if err != nil {
			log.Fatal(err)
		}

		service.StartMessaging(dht, stats, nodeType, config.ParcelSize, ctx)
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

	wg.Wait()
	cancel()

	// if nodeType != "builder" {
	// 	// ? Wait for a couple of seconds to make sure bootstrap peer is up and running
	// 	time.Sleep(5 * time.Second)
	// }

	// ctx, cancel := context.WithCancel(context.Background())

	// h, err := NewHost(ctx, config.Seed, config.Port)
	// if err != nil {
	// 	fmt.Printf("NewHost() failed\n")
	// 	log.Fatal(err)
	// }

	// if nodeType == "builder" {
	// 	log.Printf("[%s] Peer ID: %s\n", h.ID().Pretty()[:5], h.ID().Pretty())
	// }

	// log.Printf("[%s] %s Host created with ID: %s\n", h.ID()[:5].Pretty(), nodeType, h.ID()[:5].Pretty())

	// dht, err := NewDHT(ctx, h, nodeType)
	// if err != nil {
	// 	log.Printf("Error creating dht\n")
	// 	log.Fatal(err)
	// }

	// // ? Connect to bootstrap peers
	// if nodeType != "builder" {
	// 	for _, peerAddr := range config.DiscoveryPeers {
	// 		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)

	// 		if err := h.Connect(ctx, *peerinfo); err != nil {
	// 			log.Print()
	// 			log.Printf("Error connecting to bootstrap node %q: %-v", peerinfo, err)
	// 			log.Printf("peerinfo: %s\n", peerinfo)
	// 			log.Printf("peerinfo.ID: %s\n", peerinfo.ID)
	// 			log.Printf("peerinfo.Addrs: %s\n", peerinfo.Addrs)
	// 			log.Printf("err: %s\n", err)
	// 			log.Print()
	// 		}
	// 	}
	// }

	// peers, err := dht.GetClosestPeers(ctx, string(h.ID()))
	// if err != nil {
	// 	log.Printf("Error getting closest peers\n")
	// 	log.Fatal(err)
	// }
	// log.Printf("Closest peers: %d\n", len(peers))

	// go Discover(ctx, h, dht, config.Rendezvous)

	// service := NewService(h, protocol.ID(config.ProtocolID))
	// err = service.SetupRPC()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // ? Create a timer that runs for x seconds
	// timer := time.NewTimer(time.Duration(config.Duration) * time.Second)

	// go func() {
	// 	service.StartMessaging(dht, stats, nodeType, config.ParcelSize, ctx)
	// }()

	// <-timer.C

	// if err := h.Close(); err != nil {
	// 	panic(err)
	// }
	// os.Exit(0)

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

	headers := []string{"Total PUT messages", "Total failed PUTs", "Total GET messages", "Total failed GETs", "Total successful GETs"}
	rows := [][]string{
		{strconv.Itoa(stats.TotalPutMessages), strconv.Itoa(stats.TotalFailedPuts), strconv.Itoa(stats.TotalGetMessages), strconv.Itoa(stats.TotalFailedGets), strconv.Itoa(stats.TotalSuccessGets)},
	}

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
	for i := 0; i < len(stats.SeedingLatencies) || i < len(stats.PutLatencies) || i < len(stats.GetLatencies) || i < len(stats.GetHops); i++ {
		var row []string

		if i < len(stats.SeedingLatencies) {
			row = append(row, strconv.FormatInt(stats.SeedingLatencies[i].Microseconds(), 10))
		} else {
			row = append(row, "")
		}

		if i < len(stats.PutLatencies) {
			row = append(row, strconv.FormatInt(stats.PutLatencies[i].Microseconds(), 10))
		} else {
			row = append(row, "")
		}

		if i < len(stats.GetLatencies) {
			row = append(row, strconv.FormatInt(stats.GetLatencies[i].Microseconds(), 10))
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

		if i < len(stats.GetHops) {
			row = append(row, strconv.Itoa(stats.GetHops[i]))
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

	headers := []string{"Block Seeding Duration (us)", "PUT latencies (us)", "GET latencies (us)", "Row Sampling Latencies (us)", "Col Sampling Latencies (us)", "Random Sampling Latencies (us)", "GET hops"}
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

func waitForBuilder(wg *sync.WaitGroup, discoveryPeers addrList, h host.Host) {
	defer wg.Done()

	// ? Wait for a couple of seconds to make sure bootstrap peer is up and running
	time.Sleep(5 * time.Second)

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
			// log.Printf("[%s] Connected to bootstrap node %q", h.ID().Pretty()[:5], peerinfo)
			return
		}
	}

	log.Printf("Could not connect to any bootstrap nodes...")

}

func startBuilder(wg *sync.WaitGroup, ctx context.Context, seed int64, port int) {
	h, err := NewHost(ctx, seed, port)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("[B - %s] Builder created with ID: %s\n", h.ID().Pretty()[:5], h.ID().Pretty())

	select {}

}
