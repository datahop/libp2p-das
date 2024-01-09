package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
   "github.com/libp2p/go-libp2p/core/host"
   "github.com/libp2p/go-libp2p/core/peer"
   "github.com/libp2p/go-libp2p/core/protocol"
	rpc "github.com/libp2p/go-libp2p-gorpc"
	// dht "./vendor/go-libp2p-kad-dht"
)

type Service struct {
	rpcServer *rpc.Server
	rpcClient *rpc.Client
	host      host.Host
	protocol  protocol.ID
}

type Parcel struct {
	StartingIndex int
	IsRow         bool
	SampleCount   int
}

func contains(arr []int, val int) bool {
	for _, a := range arr {
		if a == val {
			return true
		}
	}
	return false
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

func SplitSamplesIntoParcels(RowCount, parcelSize int, parcelType string) []Parcel {
	TotalSamplesCount := RowCount * RowCount
	parcels := make([]Parcel, 0)

	TotalParcelsRequiredCount := (RowCount * (RowCount / parcelSize)) * 2

	// Split the samples into row parcels
	for i := 0; i < TotalSamplesCount; i += parcelSize {
		parcel := Parcel{
			StartingIndex: i,
			SampleCount:   parcelSize,
			IsRow:         true,
		}
		parcels = append(parcels, parcel)
	}

	// Split the samples into column parcels
	rowID := 0
	colID := 0
	for colID < RowCount {
		for i := 0; i < parcelSize; i++ {
			parcelID := rowID*RowCount + colID
			parcel := Parcel{
				StartingIndex: parcelID,
				SampleCount:   parcelSize,
				IsRow:         false,
			}
			if i == 0 {
				parcels = append(parcels, parcel)
			}

			rowID++

			if rowID >= RowCount {
				rowID = 0
				colID++
			}
		}
	}

	if parcelType == "all" {

		if len(parcels) != TotalParcelsRequiredCount {
			log.Printf("TotalParcelsRequiredCount of %d does not match the number of parcels of %d", TotalParcelsRequiredCount, len(parcels))
			panic("TotalParcelsRequiredCount does not match the number of parcels")
		}

		return parcels

	} else if parcelType == "row" {

		rowParcels := make([]Parcel, 0)
		for _, parcel := range parcels {
			if parcel.IsRow {
				rowParcels = append(rowParcels, parcel)
			}
		}

		if len(rowParcels) != TotalParcelsRequiredCount/2 {
			log.Printf("TotalParcelsRequiredCount of %d does not match the number of parcels of %d", TotalParcelsRequiredCount/2, len(rowParcels))
			panic("TotalParcelsRequiredCount does not match the number of parcels")
		}

		return rowParcels

	} else if parcelType == "col" {

		colParcels := make([]Parcel, 0)
		for _, parcel := range parcels {
			if !parcel.IsRow {
				colParcels = append(colParcels, parcel)
			}
		}
		if len(colParcels) != TotalParcelsRequiredCount/2 {
			log.Printf("TotalParcelsRequiredCount of %d does not match the number of parcels of %d", TotalParcelsRequiredCount/2, len(colParcels))
			panic("TotalParcelsRequiredCount does not match the number of parcels")
		}
		return colParcels

	}

	if len(parcels) != TotalParcelsRequiredCount {
		log.Printf("TotalParcelsRequiredCount of %d does not match the number of parcels of %d", TotalParcelsRequiredCount, len(parcels))
		panic("TotalParcelsRequiredCount does not match the number of parcels")
	}

	return parcels

}

func getParcelCounts(parcels []Parcel) (int, int) {
	rowParcelsCount := 0
	colParcelsCount := 0

	for _, parcel := range parcels {
		if parcel.IsRow {
			rowParcelsCount++
		} else {
			colParcelsCount++
		}
	}

	return rowParcelsCount, colParcelsCount
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

func (s *Service) StartMessaging(h host.Host, dht *dht.IpfsDHT, stats *Stats, peerType string, parcelSize int, ctx context.Context, exp_duration int) {

	if h == nil {
		panic("Host is nil")
	}
	if dht == nil {
		panic("DHT is nil")
	}
	if stats == nil {
		panic("Stats is nil")
	}
	if ctx == nil {
		panic("Context is nil")
	}

	const ROW_COUNT = 512 // ? ROW_COUNTxROW_COUNT matrix
	const TOTAL_BLOCK_COUNT = 3
	const BLOCK_TIME_SEC = 12

	// Check if ROW_COUNT is divisible by parcelSize
	if ROW_COUNT%parcelSize != 0 {
		log.Printf("ROW_COUNT of %d is not divisible by parcelSize of %d", ROW_COUNT, parcelSize)
		return
	}

   expeDurationTicker := time.NewTicker(time.Duration(exp_duration) * time.Second)
   defer expeDurationTicker.Stop()
   blockID := 0

   pub, err := CreatePubSub(h, ctx)
   if err != nil {
      log.Println("Error creating pubSub:", err)
      return
   }

   if peerType == "builder" {

		for len(dht.RoutingTable().ListPeers()) == 0 {
			log.Printf("[B - %s] Waiting for peers to join...\n", s.host.ID()[0:5])
			time.Sleep(time.Second)
		}

      // TODO add exp_duration as a parameter
		blockTicker := time.NewTicker(BLOCK_TIME_SEC * time.Second)
      defer blockTicker.Stop()
      for {
         select {
         case <-expeDurationTicker.C:
            log.Println("Experiment time exceeded")
            //finished <- true
            return

         case <-blockTicker.C:
            pub.HeaderPublish(blockID)
            blockID += 1
            go StartSeedingBlock(blockID, ROW_COUNT, parcelSize, s, ctx, stats, dht)
            //TODO add a mutex to make currBlock thread-safe
         default:
         }
      }

	} else if peerType == "validator" {
      go pub.readLoop()
      for {
         select {
         case <-expeDurationTicker.C:
            log.Println("Experiment time exceded")
            //finished <- true
            return
         case m := <-pub.messages:
            //log.Printf("Got a message %s", msg)
            blockID = m.BlockID
            go StartValidatorSampling(blockID, ROW_COUNT, parcelSize, s, ctx, stats, dht)

         default:
         }
      }

	} else if peerType == "nonvalidator" {
      go pub.readLoop()
      for {
         select {
         case <-expeDurationTicker.C:
            log.Println("Experiment time exceded")
            //finished <- true
            return
         case m := <-pub.messages:
            //log.Printf("Got a message %s", msg)
            blockID = m.BlockID
            go StartRegularSampling(blockID, ROW_COUNT, parcelSize, s, ctx, stats, dht)
         default:
         }
      }


	} else {
		panic("Peer type not recognized: " + peerType)
	}
}

func sortParcelsByStartingIndex(parcels []Parcel) {
	// We'll use the sort.Slice function to sort the parcels based on StartingIndex
	sort.Slice(parcels, func(i, j int) bool {
		return parcels[i].StartingIndex < parcels[j].StartingIndex
	})
}

func pickRandomParcels(parcels []Parcel, requiredCount int) []Parcel {

	// Create a new source of randomness based on the current time
	source := rand.NewSource(time.Now().UnixNano())

	// Create a new random number generator using the source
	randomGenerator := rand.New(source)

	sortParcelsByStartingIndex(parcels)

	// Check if the requiredCount is greater than the total number of parcels
	if requiredCount >= len(parcels) {
		return parcels // Return all parcels if requiredCount is greater or equal
	}

	// Create a slice to store the selected random parcels
	randomParcels := make([]Parcel, requiredCount)

	for i := 0; i < requiredCount; i++ {
		randomIndex := randomGenerator.Intn(len(parcels))
		randomParcels[i] = parcels[randomIndex]
	}

	if len(randomParcels) != requiredCount {
		log.Printf("Random parcel count of %d does not match the required count of %d", len(randomParcels), requiredCount)
		panic("Random parcel count does not match the required count")
	}

	return randomParcels
}
