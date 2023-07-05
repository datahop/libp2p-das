package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
)

type Config struct {
	Port           int
	ProtocolID     string
	Rendezvous     string
	Seed           int64
	DiscoveryPeers addrList
	Debug          bool
	Duration       int
}

type Stats struct {
	TotalPutMessages int
	TotalFailedPuts int
	TotalGetMessages int
	TotalFailedGets  int
	TotalSuccessGets int
	// Array of latencies for puts
	PutLatencies []time.Duration
	// Array of latencies for gets
	GetLatencies []time.Duration
	// Array of hops for gets
	GetHops []int
}

func colorize(word string, colorName string) string {
	var c *color.Color
	switch colorName {
	case "red":
		c = color.New(color.FgRed)
	case "green":
		c = color.New(color.FgGreen)
	case "yellow":
		c = color.New(color.FgYellow)
	case "blue":
		c = color.New(color.FgBlue)
	default:
		c = color.New(color.Reset)
	}

	return c.Sprint(word)
}

func main() {
	config := Config{}
	stats := &Stats{}

	var debugMode bool = false

	flag.StringVar(&config.Rendezvous, "rendezvous", "/echo", "")
	flag.Int64Var(&config.Seed, "seed", 0, "Seed value for generating a PeerID, 0 is random")
	flag.Var(&config.DiscoveryPeers, "peer", "Peer multiaddress for peer discovery")
	flag.StringVar(&config.ProtocolID, "protocolid", "/p2p/rpc", "")
	flag.IntVar(&config.Port, "port", 0, "")
	flag.IntVar(&config.Duration, "duration", 15, "How long to run the test for (in seconds).")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode - see more messages about what is happening.")
	flag.Parse()

	if debugMode {
		log.Printf("Running libp2p-das with the following config:\n")
		log.Printf("\tRendezvous: %s\n", config.Rendezvous)
		log.Printf("\tSeed: %d\n", config.Seed)
		log.Printf("\tDiscoveryPeers: %s\n", config.DiscoveryPeers)
		log.Printf("\tProtocolID: %s\n", config.ProtocolID)
		log.Printf("\tPort: %d\n\n", config.Port)
	}

	ctx, cancel := context.WithCancel(context.Background())

	h, peerType, err := NewHost(ctx, config.Seed, config.Port)

	if err != nil {
		fmt.Printf("NewHost() failed\n")
		log.Fatal(err)
	}

	log.Print(colorize("Created Host ID: "+h.ID()[0:7].Pretty()+"\n", "white"))

	log.Printf("Connect to me on:")
	for _, addr := range h.Addrs() {
		log.Printf("  %s/p2p/%s", addr, h.ID().Pretty())
	}

	dht, err := NewDHT(ctx, h, config.DiscoveryPeers)
	if err != nil {
		log.Printf("Error creating dht\n")
		log.Fatal(err)
	}

	go Discover(ctx, h, dht, config.Rendezvous)

	service := NewService(h, protocol.ID(config.ProtocolID))
	err = service.SetupRPC()
	if err != nil {
		log.Fatal(err)
	}

	// Create a timer that runs for x seconds
	timer := time.NewTimer(time.Duration(config.Duration) * time.Second)

	// Start the messaging service in a separate goroutine
	go func() {
		service.StartMessaging(dht, stats, peerType, ctx)
	}()

	// Wait for the timer to expire
	<-timer.C
	log.Printf("Timer expired, shutting down...\n")

	if filename, err := writeTotalStatsToFile(stats, h, peerType); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("[%s] Total Stats written to %s\n", h.ID()[0:5].Pretty(), filename)
	}

	if filename, err := writeLatencyStatsToFile(stats, h, peerType); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("[%s] Latencies written to %s\n", h.ID()[0:5].Pretty(), filename)
	}

	// Cancel the context and exit
	cancel()

	if err := h.Close(); err != nil {
		panic(err)
	}
	os.Exit(0)

}

func writeTotalStatsToFile(stats *Stats, h host.Host, peerType string) (string, error) {
	filename := h.ID()[0:10].Pretty() + "_total_stats_" + peerType + ".csv"

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

func writeLatencyStatsToFile(stats *Stats, h host.Host, peerType string) (string, error) {
	filename := h.ID()[0:10].Pretty() + "_latency_stats_" + peerType + ".csv"

	// Convert latencies and hops to rows
	var latencyRows [][]string
	for i := 0; i < len(stats.PutLatencies) || i < len(stats.GetLatencies) || i < len(stats.GetHops); i++ {
		var row []string
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

	headers := []string{"PUT latencies (us)", "GET latencies (us)", "GET hops"}
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
