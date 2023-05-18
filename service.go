package main

import (
    "context"
    "fmt"
    "time"

    "github.com/libp2p/go-libp2p-core/host"
    "github.com/libp2p/go-libp2p-core/peer"
    "github.com/libp2p/go-libp2p-core/protocol"
    rpc "github.com/libp2p/go-libp2p-gorpc"
    dht "github.com/libp2p/go-libp2p-kad-dht"
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

func (s *Service) StartMessaging(dht *dht.IpfsDHT, ctx context.Context) {
    // Blob is 42KB
    var samples []byte = make([]byte, 42000)

    ticker := time.NewTicker(time.Second * 1)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.counter++
            s.Echo(dht, samples, ctx)
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
