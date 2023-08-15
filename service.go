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

type Parcel struct {
	StartingIndex int
	IsRow         bool
	SampleCount   int
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

		return parcels

	} else if parcelType == "row" {

		rowParcels := make([]Parcel, 0)
		for _, parcel := range parcels {
			if parcel.IsRow {
				rowParcels = append(rowParcels, parcel)
			}
		}
		return rowParcels

	} else if parcelType == "col" {

		colParcels := make([]Parcel, 0)
		for _, parcel := range parcels {
			if !parcel.IsRow {
				colParcels = append(colParcels, parcel)
			}
		}
		return colParcels

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

// func (s *Service) StartMessaging(dht *dht.IpfsDHT, stats *Stats, peerType string, parcelSize int, ctx context.Context) {
// 	ticker := time.NewTicker(time.Millisecond * 100)
// 	defer ticker.Stop()

// 	const RowCount = 512
// 	const TotalSamplesCount = RowCount * RowCount
// 	const TotalBlocksCount = 10

// 	blockID := 0

// 	parcels := SplitSamplesIntoParcels(RowCount, parcelSize)

// 	var parcelsReceived []Parcel

// 	// ! Validator Variables:
// 	colParcelsReceivedCount := 0
// 	rowParcelsReceivedCount := 0
// 	randomParcelsReceivedCount := 0

// 	rowSamplingStartTime := time.Now()
// 	colSamplingStartTime := time.Now()
// 	randomSamplingStartTime := time.Now()

// 	rowSamplingLatencyRecorded := false
// 	colSamplingLatencyRecorded := false
// 	randomSamplingLatencyRecorded := false

// 	// ? Find out how many parcels are needed to make up at least half the row
// 	halfRowCount := RowCount / 2

// 	rowParcelsNeededCount := halfRowCount/parcelSize + 1
// 	// ? 2 rows
// 	rowParcelsNeededCount *= 2

// 	colParcelsNeededCount := halfRowCount/parcelSize + 1
// 	// ? 2 columns
// 	colParcelsNeededCount *= 2

// 	// ? 75 random samples too
// 	randomParcelsNeededCount := 75

// 	totalParcelsNeededCount := rowParcelsNeededCount + colParcelsNeededCount + randomParcelsNeededCount

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case <-ticker.C:

// 			if peerType == "builder" {

// 				// ? If all parcels are sent, go to the next block
// 				if len(parcels) == 0 && blockID < TotalBlocksCount {
// 					blockID += 1
// 					parcels = SplitSamplesIntoParcels(RowCount, parcelSize)
// 				}

// 				// ? If all blocks are sent, stop
// 				if blockID >= TotalBlocksCount {
// 					continue
// 				}

// 				parcelToSend := parcels[0]

// 				// ? Get the samples - 512 bytes per sample
// 				parcelSamplesToSend := make([]byte, parcelToSend.SampleCount*512)

// 				peers := FilterSelf(s.host.Peerstore().Peers(), s.host.ID())
// 				dhtPeers := FilterSelf(dht.RoutingTable().ListPeers(), s.host.ID())

// 				// ? No peers found, skip
// 				if len(peers) == 0 && len(dhtPeers) == 0 {
// 					continue
// 				}

// 				// ? Manually add peers to routing table
// 				if len(peers) == 0 || len(dhtPeers) == 0 {
// 					for _, p := range peers {
// 						_, err := dht.RoutingTable().TryAddPeer(p, false, true)
// 						if err != nil {
// 							log.Printf("Failed to add peer %s : %s\n", p[0:5].Pretty(), err.Error())
// 						}
// 					}
// 				}

// 				startTime := time.Now()

// 				parcelType := "row"
// 				if !parcelToSend.IsRow {
// 					parcelType = "col"
// 				}

// 				// ? Put parcel samples into DHT
// 				putErr := dht.PutValue(ctx, "/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(parcelToSend.StartingIndex), parcelSamplesToSend)

// 				if putErr != nil {
// 					stats.TotalFailedPuts += 1
// 					stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))

// 					log.Print("[BUILDER\t\t" + s.host.ID()[0:5].Pretty() + "] PutValue() Error: " + putErr.Error())
// 					log.Printf("[BUILDER\t\t"+s.host.ID()[0:5].Pretty()+"]: DHT Peers: %d\n", len(dht.RoutingTable().ListPeers()))

// 				} else {
// 					stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))
// 					stats.TotalPutMessages += 1

// 					// ? Remove first parcel from the list
// 					if len(parcels) >= 1 {
// 						parcels = parcels[1:]
// 					}

// 					log.Print(
// 						printOperation(
// 							"[BUILDER\t"+s.host.ID()[0:5].Pretty()+"]",
// 							true,
// 							true,
// 							parcelToSend,
// 							blockID,
// 							RowCount,
// 							len(SplitSamplesIntoParcels(RowCount, parcelSize))-len(parcels),
// 							len(SplitSamplesIntoParcels(RowCount, parcelSize)),
// 							-1,
// 						),
// 					)

// 				}

// 			} else if peerType == "validator" {

// 				// ? If all parcels are received, go to the next block
// 				if len(parcelsReceived) >= totalParcelsNeededCount && blockID < TotalBlocksCount && blockID >= 0 {
// 					blockID += 1
// 					colParcelsReceivedCount = 0
// 					rowParcelsReceivedCount = 0
// 					randomParcelsReceivedCount = 0

// 					rowSamplingStartTime = time.Now()
// 					colSamplingStartTime = time.Now()
// 					randomSamplingStartTime = time.Now()

// 					rowSamplingLatencyRecorded = false
// 					colSamplingLatencyRecorded = false
// 					randomSamplingLatencyRecorded = false

// 					parcelsReceived = make([]Parcel, 0)
// 				}

// 				// ! Row Parcel Sampling

// 				// ? Get row parcel
// 				if rowParcelsReceivedCount < rowParcelsNeededCount {
// 					// ? Pick a random row parcel that has not been received yet
// 					parcelID := rand.Intn(len(parcels))
// 					for _, p := range parcelsReceived {
// 						// ? If the parcel ID and type has already been received, record it and continue
// 						if p.StartingIndex == parcels[parcelID].StartingIndex && p.IsRow == parcels[parcelID].IsRow {
// 							rowParcelsReceivedCount += 1
// 							parcelsReceived = append(parcelsReceived, parcels[parcelID])
// 							log.Print(
// 								printOperation(
// 									"[VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
// 									false,
// 									true,
// 									parcels[parcelID],
// 									blockID,
// 									RowCount,
// 									rowParcelsReceivedCount,
// 									rowParcelsNeededCount,
// 									-1,
// 								),
// 							)
// 							continue
// 						}
// 					}

// 					// ? Get the parcel
// 					parcelToGet := parcels[parcelID]
// 					parcelType := "row"
// 					if !parcelToGet.IsRow {
// 						parcelType = "col"
// 					}
// 					startTime := time.Now()
// 					_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(parcelToGet.StartingIndex))
// 					if err != nil {
// 						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
// 						stats.TotalFailedGets += 1
// 						stats.TotalGetMessages += 1
// 						stats.GetHops = append(stats.GetHops, hops)
// 					} else {
// 						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
// 						stats.TotalGetMessages += 1
// 						stats.GetHops = append(stats.GetHops, hops)

// 						rowParcelsReceivedCount += 1
// 						parcelsReceived = append(parcelsReceived, parcelToGet)

// 						log.Print(
// 							printOperation(
// 								"[VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
// 								false,
// 								true,
// 								parcelToGet,
// 								blockID,
// 								RowCount,
// 								rowParcelsReceivedCount,
// 								rowParcelsNeededCount,
// 								-1,
// 							),
// 						)
// 					}
// 				} else {
// 					// log.Print("[VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] Row Sampling Done.")
// 					if !rowSamplingLatencyRecorded {
// 						stats.RowSamplingLatencies = append(stats.RowSamplingLatencies, time.Since(rowSamplingStartTime))
// 						rowSamplingLatencyRecorded = true
// 					}
// 				}

// 				// ! Col Parcel Sampling

// 				// ? Get col parcel
// 				if colParcelsReceivedCount < colParcelsNeededCount {
// 					// ? Pick a random col parcel that has not been received yet
// 					parcelID := rand.Intn(len(parcels))
// 					for _, p := range parcelsReceived {
// 						// ? If the parcel ID and type has already been received, record it and continue
// 						if p.StartingIndex == parcels[parcelID].StartingIndex && p.IsRow == parcels[parcelID].IsRow {
// 							colParcelsReceivedCount += 1
// 							parcelsReceived = append(parcelsReceived, parcels[parcelID])
// 							log.Print(
// 								printOperation(
// 									"[VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
// 									false,
// 									true,
// 									parcels[parcelID],
// 									blockID,
// 									RowCount,
// 									colParcelsReceivedCount,
// 									colParcelsNeededCount,
// 									-1,
// 								),
// 							)
// 							continue
// 						}
// 					}

// 					// ? Get the parcel
// 					parcelToGet := parcels[parcelID]
// 					if parcelToGet.IsRow {
// 						continue
// 					}
// 					startTime := time.Now()
// 					_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(blockID)+"/col/"+fmt.Sprint(parcelToGet.StartingIndex))
// 					if err != nil {
// 						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
// 						stats.TotalFailedGets += 1
// 						stats.TotalGetMessages += 1
// 						stats.GetHops = append(stats.GetHops, hops)
// 					} else {
// 						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
// 						stats.TotalGetMessages += 1
// 						stats.GetHops = append(stats.GetHops, hops)

// 						colParcelsReceivedCount += 1
// 						parcelsReceived = append(parcelsReceived, parcelToGet)

// 						log.Print(
// 							printOperation(
// 								"[VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
// 								false,
// 								true,
// 								parcelToGet,
// 								blockID,
// 								RowCount,
// 								colParcelsReceivedCount,
// 								colParcelsNeededCount,
// 								-1,
// 							),
// 						)
// 					}
// 				} else {
// 					// log.Print("[VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] Col Sampling Done.")
// 					if !colSamplingLatencyRecorded {
// 						stats.ColSamplingLatencies = append(stats.ColSamplingLatencies, time.Since(colSamplingStartTime))
// 						colSamplingLatencyRecorded = true
// 					}
// 				}

// 				// ! 75 Random Parcel Sampling

// 				if randomParcelsReceivedCount < randomParcelsNeededCount {
// 					randomSampleID := rand.Intn(TotalSamplesCount)

// 					// ? Find the parcel that contains the sample
// 					parcelContainingSample := Parcel{}
// 					for _, parcel := range parcels {
// 						parcelSampleIDs := make([]int, parcel.SampleCount)

// 						if parcel.IsRow {
// 							for i := 0; i < parcel.SampleCount; i++ {
// 								parcelSampleIDs[i] = parcel.StartingIndex + i
// 							}
// 						} else {
// 							for i := 0; i < parcel.SampleCount; i++ {
// 								parcelSampleIDs[i] = parcel.StartingIndex + i*RowCount
// 							}
// 						}

// 						// ? If the parcel contains the sample, break
// 						found := false
// 						for _, id := range parcelSampleIDs {
// 							if id == randomSampleID {
// 								found = true
// 								break
// 							}
// 						}

// 						if found {
// 							parcelContainingSample = parcel
// 							break
// 						}

// 					}

// 					// ? If the parcel is empty, continue to next loop
// 					if parcelContainingSample == (Parcel{}) {
// 						continue
// 					}

// 					// ? Check if the parcelContainingSample is in parcelsReceived
// 					parcelAlreadyReceived := false
// 					for _, p := range parcelsReceived {
// 						if p.StartingIndex == parcelContainingSample.StartingIndex && p.IsRow == parcelContainingSample.IsRow {
// 							parcelAlreadyReceived = true
// 							break
// 						}
// 					}

// 					// ? If the parcel is already received, continue to next loop
// 					if parcelAlreadyReceived {
// 						randomParcelsReceivedCount += 1
// 						parcelsReceived = append(parcelsReceived, parcelContainingSample)
// 						log.Print(
// 							printOperation(
// 								"[VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
// 								false,
// 								true,
// 								parcelContainingSample,
// 								blockID,
// 								RowCount,
// 								randomParcelsReceivedCount,
// 								randomParcelsNeededCount,
// 								randomSampleID,
// 							),
// 						)
// 						continue
// 					}

// 					// ? Get the parcel
// 					parcelToGet := parcelContainingSample
// 					parcelType := "row"
// 					if !parcelToGet.IsRow {
// 						parcelType = "col"
// 					}
// 					startTime := time.Now()
// 					_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(parcelToGet.StartingIndex))
// 					if err != nil {
// 						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
// 						stats.TotalFailedGets += 1
// 						stats.TotalGetMessages += 1
// 						stats.GetHops = append(stats.GetHops, hops)
// 					} else {
// 						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
// 						stats.TotalGetMessages += 1
// 						stats.GetHops = append(stats.GetHops, hops)

// 						randomParcelsReceivedCount += 1
// 						parcelsReceived = append(parcelsReceived, parcelToGet)

// 						log.Print(
// 							printOperation(
// 								"[VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
// 								false,
// 								true,
// 								parcelToGet,
// 								blockID,
// 								RowCount,
// 								randomParcelsReceivedCount,
// 								randomParcelsNeededCount,
// 								randomSampleID,
// 							),
// 						)
// 					}

// 					if randomParcelsReceivedCount == randomParcelsNeededCount {
// 						if !randomSamplingLatencyRecorded {
// 							stats.RandomSamplingLatencies = append(stats.RandomSamplingLatencies, time.Since(randomSamplingStartTime))
// 							randomSamplingLatencyRecorded = true
// 						}
// 					}

// 				}

// 			} else if peerType == "nonvalidator" {

// 				// ? Reset timers if all parcels are received
// 				if len(parcelsReceived) == 0 && blockID > 0 {
// 					randomSamplingStartTime = time.Now()
// 					randomSamplingLatencyRecorded = false
// 					randomParcelsReceivedCount = 0
// 				}

// 				// ! 75 Random Parcel Sampling

// 				// ? If all parcels are received, go to the next block
// 				if len(parcelsReceived) >= randomParcelsNeededCount && blockID < TotalBlocksCount {
// 					blockID += 1
// 					randomParcelsReceivedCount = 0
// 					randomSamplingStartTime = time.Now()
// 					randomSamplingLatencyRecorded = false
// 					parcelsReceived = make([]Parcel, 0)
// 				}

// 				if randomParcelsReceivedCount < randomParcelsNeededCount {
// 					randomSampleID := rand.Intn(TotalSamplesCount)

// 					// ? Find the parcel that contains the sample
// 					parcelContainingSample := Parcel{}
// 					for _, parcel := range parcels {
// 						parcelSampleIDs := make([]int, parcel.SampleCount)

// 						if parcel.IsRow {
// 							for i := 0; i < parcel.SampleCount; i++ {
// 								parcelSampleIDs[i] = parcel.StartingIndex + i
// 							}
// 						} else {
// 							for i := 0; i < parcel.SampleCount; i++ {
// 								parcelSampleIDs[i] = parcel.StartingIndex + i*RowCount
// 							}
// 						}

// 						// ? If the parcel contains the sample, break
// 						found := false
// 						for _, id := range parcelSampleIDs {
// 							if id == randomSampleID {
// 								found = true
// 								break
// 							}
// 						}

// 						if found {
// 							parcelContainingSample = parcel
// 							break
// 						}

// 					}

// 					// ? If the parcel is empty, continue to next loop
// 					if parcelContainingSample == (Parcel{}) {
// 						continue
// 					}

// 					// ? Check if the parcelContainingSample is in parcelsReceived
// 					parcelAlreadyReceived := false
// 					for _, p := range parcelsReceived {
// 						if p.StartingIndex == parcelContainingSample.StartingIndex && p.IsRow == parcelContainingSample.IsRow {
// 							parcelAlreadyReceived = true
// 							break
// 						}
// 					}
// 					// ? If the parcel is already received, continue to next loop
// 					if parcelAlreadyReceived {
// 						randomParcelsReceivedCount += 1
// 						parcelsReceived = append(parcelsReceived, parcelContainingSample)
// 						log.Print(
// 							printOperation(
// 								"[NON VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
// 								false,
// 								true,
// 								parcelContainingSample,
// 								blockID,
// 								RowCount,
// 								randomParcelsReceivedCount,
// 								randomParcelsNeededCount,
// 								randomSampleID,
// 							),
// 						)
// 						continue
// 					}

// 					// ? Get the parcel
// 					parcelToGet := parcelContainingSample
// 					parcelType := "row"
// 					if !parcelToGet.IsRow {
// 						parcelType = "col"
// 					}
// 					startTime := time.Now()
// 					_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(parcelToGet.StartingIndex))

// 					if err != nil {
// 						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
// 						stats.TotalFailedGets += 1
// 						stats.TotalGetMessages += 1
// 						stats.GetHops = append(stats.GetHops, hops)
// 					} else {
// 						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
// 						stats.TotalGetMessages += 1
// 						stats.GetHops = append(stats.GetHops, hops)

// 						randomParcelsReceivedCount += 1
// 						parcelsReceived = append(parcelsReceived, parcelToGet)

// 						log.Print(
// 							printOperation(
// 								"[NON VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
// 								false,
// 								true,
// 								parcelToGet,
// 								blockID,
// 								RowCount,
// 								randomParcelsReceivedCount,
// 								randomParcelsNeededCount,
// 								randomSampleID,
// 							),
// 						)
// 					}

// 					if randomParcelsReceivedCount == randomParcelsNeededCount {
// 						if !randomSamplingLatencyRecorded {
// 							stats.RandomSamplingLatencies = append(stats.RandomSamplingLatencies, time.Since(randomSamplingStartTime))
// 							randomSamplingLatencyRecorded = true
// 						}
// 					}
// 				}
// 			}

// 		}
// 	}
// }

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

func (s *Service) StartMessaging(dht *dht.IpfsDHT, stats *Stats, peerType string, parcelSize int, ctx context.Context) {

	const RowCount = 512
	const TotalBlocksCount = 3

	if peerType == "builder" {

		peerName := "[BUILDER\t\t" + s.host.ID()[0:5].Pretty() + "]"

		blockID := 0
		for blockID < TotalBlocksCount {

			// ! Network setup

			peers := FilterSelf(s.host.Peerstore().Peers(), s.host.ID())
			dhtPeers := FilterSelf(dht.RoutingTable().ListPeers(), s.host.ID())

			// ? No peers found, skip
			if len(peers) == 0 && len(dhtPeers) == 0 {
				continue
			}

			// ? Manually add peers to routing table
			if len(peers) == 0 || len(dhtPeers) == 0 {
				for _, p := range peers {
					_, err := dht.RoutingTable().TryAddPeer(p, false, true)
					if err != nil {
						log.Printf("Failed to add peer %s : %s\n", p[0:5].Pretty(), err.Error())
					}
				}
			}

			// ! Seeding

			seedingTime := time.Now()

			parcels := SplitSamplesIntoParcels(RowCount, parcelSize, "all")

			for len(parcels) > 0 {

				randomIndex := 0
				if len(parcels) > 1 {
					randomIndex = rand.Intn(len(parcels) - 1)
				}

				randomParcel := parcels[randomIndex]

				parcelSamplesToSend := make([]byte, randomParcel.SampleCount*512)

				parcelType := "row"
				if !randomParcel.IsRow {
					parcelType = "col"
				}

				putStartTime := time.Now()
				putErr := dht.PutValue(
					ctx,
					"/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(randomParcel.StartingIndex),
					parcelSamplesToSend,
				)

				if putErr != nil {
					stats.TotalFailedPuts += 1
					stats.TotalPutMessages += 1
				} else {
					// log.Printf(
					// 	peerName+" B%d: [%d] PUT parcel %s %d took %.2f seconds.\n",
					// 	blockID,
					// 	len(parcels),
					// 	parcelType,
					// 	randomParcel.StartingIndex,
					// 	time.Since(putStartTime).Seconds(),
					// )
					stats.PutLatencies = append(stats.PutLatencies, time.Since(putStartTime))
					stats.TotalPutMessages += 1

					parcels = append(parcels[:randomIndex], parcels[randomIndex+1:]...)

				}

				if len(parcels) == 0 {
					stats.SeedingLatencies = append(stats.SeedingLatencies, time.Since(seedingTime))
					log.Printf(peerName+" B%d: Seeding took %.2f seconds.\n", blockID, time.Since(seedingTime).Seconds())
				}

			}

			blockID += 1
		}

	} else if peerType == "validator" {

		peerName := "[VALIDATOR\t\t" + s.host.ID()[0:5].Pretty() + "]"

		blockID := 0

		for blockID < TotalBlocksCount {
			startTime := time.Now()
			rowSamplingDurationMicrosec := int64(0)
			colSamplingDurationMicrosec := int64(0)

			rowColParcelsNeededCount := (RowCount / 2) / parcelSize
			randomParcelsNeededCount := 75

			allParcels := SplitSamplesIntoParcels(RowCount, parcelSize, "all")
			rowParcels := SplitSamplesIntoParcels(RowCount, parcelSize, "row")
			colParcels := SplitSamplesIntoParcels(RowCount, parcelSize, "col")

			randomRowParcels := pickRandomParcels(rowParcels, rowColParcelsNeededCount)
			randomColParcels := pickRandomParcels(colParcels, rowColParcelsNeededCount)
			randomParcels := pickRandomParcels(allParcels, randomParcelsNeededCount)

			allRandomParcels := append(randomRowParcels, randomColParcels...)
			allRandomParcels = append(allRandomParcels, randomParcels...)

			for len(allRandomParcels) > 0 {
				randomIndex := 0
				if len(allRandomParcels) > 1 {
					randomIndex = rand.Intn(len(allRandomParcels))
				}

				randomParcel := allRandomParcels[randomIndex]

				parcelType := "row"
				if !randomParcel.IsRow {
					parcelType = "col"
				}

				getStartTime := time.Now()
				_, hops, err := dht.GetValueHops(
					ctx,
					"/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(randomParcel.StartingIndex),
				)

				if err != nil {
					stats.TotalFailedGets += 1
					stats.TotalGetMessages += 1
				} else {
					stats.GetLatencies = append(stats.GetLatencies, time.Since(getStartTime))
					stats.TotalGetMessages += 1
					stats.TotalSuccessGets += 1
					stats.GetHops = append(stats.GetHops, hops)

					allRandomParcels = append(allRandomParcels[:randomIndex], allRandomParcels[randomIndex+1:]...)

					rowParcelCount, colParcelCount := getParcelCounts(allRandomParcels)

					if rowParcelCount == 0 && rowSamplingDurationMicrosec == 0 {
						rowSamplingDurationMicrosec = time.Since(startTime).Microseconds()
						stats.RowSamplingLatencies = append(stats.RowSamplingLatencies, time.Since(startTime))
					}

					if colParcelCount == 0 && colSamplingDurationMicrosec == 0 {
						colSamplingDurationMicrosec = time.Since(startTime).Microseconds()
						stats.ColSamplingLatencies = append(stats.ColSamplingLatencies, time.Since(startTime))
					}

				}

			}

			if len(allRandomParcels) == 0 {
				log.Printf(peerName+" B%d: All Sampling took %.2f seconds.\n", blockID, time.Since(startTime).Seconds())

				blockID += 1
			}
		}

	} else if peerType == "nonvalidator" {

		peerName := "[NON-VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "]"

		blockID := 0

		for blockID < TotalBlocksCount {
			startTime := time.Now()
			randomSamplingDurationMicrosec := int64(0)

			randomParcelsNeededCount := 75

			allParcels := SplitSamplesIntoParcels(RowCount, parcelSize, "all")

			randomParcels := pickRandomParcels(allParcels, randomParcelsNeededCount)

			for len(randomParcels) > 0 {
				randomIndex := 0
				if len(randomParcels) > 1 {
					randomIndex = rand.Intn(len(randomParcels))
				}

				randomParcel := randomParcels[randomIndex]

				parcelType := "row"
				if !randomParcel.IsRow {
					parcelType = "col"
				}

				getStartTime := time.Now()
				_, hops, err := dht.GetValueHops(
					ctx,
					"/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(randomParcel.StartingIndex),
				)

				if err != nil {
					stats.TotalFailedGets += 1
					stats.TotalGetMessages += 1
				} else {
					stats.GetLatencies = append(stats.GetLatencies, time.Since(getStartTime))
					stats.TotalGetMessages += 1
					stats.TotalSuccessGets += 1
					stats.GetHops = append(stats.GetHops, hops)

					// log.Printf(peerName+" B%d: GET parcel %s %d\n", blockID, parcelType, randomParcel.StartingIndex)

					randomParcels = append(randomParcels[:randomIndex], randomParcels[randomIndex+1:]...)

					if len(randomParcels) == 0 && randomSamplingDurationMicrosec == 0 {
						randomSamplingDurationMicrosec = time.Since(startTime).Microseconds()
						stats.RandomSamplingLatencies = append(stats.RandomSamplingLatencies, time.Since(startTime))
					}

				}

			}

			if len(randomParcels) == 0 {
				log.Printf(peerName+" B%d: All Sampling took %.2f seconds.\n", blockID, time.Since(startTime).Seconds())
				blockID += 1
			}
		}

	} else {
		panic("Peer type not recognized: " + peerType)
	}
}

func pickRandomParcels(parcels []Parcel, requiredCount int) []Parcel {
	randomParcels := make([]Parcel, 0)
	for i := 0; i < requiredCount; i++ {
		randomIndex := rand.Intn(len(parcels))
		randomParcel := parcels[randomIndex]

		// ? Check if the random parcel has already been picked
		alreadyPicked := false
		for _, p := range randomParcels {
			if p.StartingIndex == randomParcel.StartingIndex && p.IsRow == randomParcel.IsRow {
				alreadyPicked = true
				break
			}
		}

		// ? If the random parcel has not been picked, add it to the list
		if !alreadyPicked {
			randomParcels = append(randomParcels, randomParcel)
		}
	}

	return randomParcels
}
