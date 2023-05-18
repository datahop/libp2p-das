package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

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
}

func main() {
    config := Config{}

    var debugMode bool = false

    flag.StringVar(&config.Rendezvous, "rendezvous", "/echo", "")
    flag.Int64Var(&config.Seed, "seed", 0, "Seed value for generating a PeerID, 0 is random")
    flag.Var(&config.DiscoveryPeers, "peer", "Peer multiaddress for peer discovery")
    flag.StringVar(&config.ProtocolID, "protocolid", "/p2p/rpc", "")
    flag.IntVar(&config.Port, "port", 0, "")
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

    h, err := NewHost(ctx, config.Seed, config.Port)
    if err != nil {
        fmt.Printf("NewHost() failed\n")
        log.Fatal(err)
    }

    fmt.Printf("Created Host ID: %s\n", h.ID()[0:7].Pretty())

    // if debugMode {
    //  log.Printf("Created new host:\n\tID: [%s] \n\tSeed: [%d] \n\tPort: [%d]", h.ID().Pretty(), config.Seed, config.Port)
    // }

    // log.Printf("Connect to me on:")
    // for _, addr := range h.Addrs() {
    //  log.Printf("  %s/p2p/%s", addr, h.ID().Pretty())
    // }

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
    
    go service.StartMessaging(dht, ctx)

    
    if len(config.DiscoveryPeers) > 0 {
        err = dht.PutValue(ctx, "/das/hello", []byte("world"))
        if err != nil {
            log.Printf("PutValue() failed\n")
            log.Fatal(err)
        }
    } else {
        time.Sleep(3 * time.Second)
        val, err := dht.GetValue(ctx, "/das/hello")
        if err != nil {
            log.Printf("GetValue failed\n")
            log.Fatal(err)
        } else {
            fmt.Printf("GetValue returned: %s", string(val))
        }
    }

    run(h, cancel)
}

func run(h host.Host, cancel func()) {
    c := make(chan os.Signal, 1)

    signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
    <-c

    fmt.Printf("\rExiting...\n")

    cancel()

    if err := h.Close(); err != nil {
        panic(err)
    }
    os.Exit(0)
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
