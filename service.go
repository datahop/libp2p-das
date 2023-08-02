package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
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

func SplitSamplesIntoParcels(RowCount, ParcelSize int) []Parcel {
	TotalSamplesCount := RowCount * RowCount
	parcels := make([]Parcel, 0)

	// Split the samples into row parcels
	for i := 0; i < TotalSamplesCount; i += ParcelSize {
		parcel := Parcel{
			StartingIndex: i,
			SampleCount:   ParcelSize,
			IsRow:         true,
		}
		parcels = append(parcels, parcel)
	}

	// Split the samples into column parcels
	rowID := 0
	colID := 0
	for colID < RowCount {
		for i := 0; i < ParcelSize; i++ {
			parcelID := rowID*RowCount + colID
			parcel := Parcel{
				StartingIndex: parcelID,
				SampleCount:   ParcelSize,
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

	return parcels
}

func printOperation(peerIdString string, isPut bool, showParcelStatusFraction bool, parcelInvolved Parcel, blockID int, RowCount int, parcelsReceivedCount int, totalParcelsNeededCount int, randomSampleID int) string {

	output := peerIdString

	if showParcelStatusFraction {
		output += " (" + strconv.Itoa(parcelsReceivedCount) + "/" + strconv.Itoa(totalParcelsNeededCount) + ")"
	}

	if isPut {
		output += " PUT "
	} else {
		output += " GET "
	}

	parcelTypeString := colorize("COL", "green")
	lastParcelID := parcelInvolved.StartingIndex + (parcelInvolved.SampleCount-1)*RowCount
	if parcelInvolved.IsRow {
		parcelTypeString = colorize("ROW", "blue")
		lastParcelID = parcelInvolved.StartingIndex + (parcelInvolved.SampleCount - 1)
	}

	blockIDString := strconv.Itoa(blockID)

	parcelContentsString := "[" + strconv.Itoa(parcelInvolved.StartingIndex) + "..." + strconv.Itoa(lastParcelID) + "]"
	if parcelInvolved.SampleCount <= 2 {
		parcelContentsString = "[" + strconv.Itoa(parcelInvolved.StartingIndex) + ", " + strconv.Itoa(lastParcelID) + "]"
	}

	if randomSampleID != -1 {
		output += parcelTypeString + " parcel BID " + blockIDString + " SID " + strconv.Itoa(randomSampleID) + ": " + parcelContentsString
	} else {
		output += parcelTypeString + " parcel BID " + blockIDString + ": " + parcelContentsString
	}

	return output
}

func (s *Service) StartMessaging(dht *dht.IpfsDHT, stats *Stats, peerType string, ctx context.Context) {
	ticker := time.NewTicker(time.Nanosecond * 1)
	defer ticker.Stop()

	const RowCount = 512
	const TotalSamplesCount = RowCount * RowCount
	const TotalBlocksCount = 10
	const ParcelSize = 512

	blockID := 0
	currentBlockID := 0

	// ? Generate 512 x 512 IDs for each sample starting from 0 to 512^2-1
	sampleIDs := make([]int, TotalSamplesCount)
	for i := 0; i < TotalSamplesCount; i++ {
		sampleIDs[i] = i
	}

	parcels := SplitSamplesIntoParcels(RowCount, ParcelSize)
	var parcelsSent []Parcel
	var parcelsReceived []Parcel

	// ! Validator Variables:
	colParcelsReceivedCount := 0
	rowParcelsReceivedCount := 0
	randomParcelsReceivedCount := 0

	rowSamplingStartTime := time.Now()
	colSamplingStartTime := time.Now()
	randomSamplingStartTime := time.Now()

	rowSamplingLatencyRecorded := false
	colSamplingLatencyRecorded := false
	randomSamplingLatencyRecorded := false

	// ? Find out how many parcels are needed to make up at least half the row
	halfRowCount := RowCount / 2

	rowParcelsNeededCount := halfRowCount // ParcelSize + 1
	// ? 2 rows
	rowParcelsNeededCount *= 2

	colParcelsNeededCount := halfRowCount // ParcelSize + 1
	// ? 2 columns
	colParcelsNeededCount *= 2

	// ? 75 random samples too
	randomParcelsNeededCount := 75

	totalParcelsNeededCount := rowParcelsNeededCount + colParcelsNeededCount + randomParcelsNeededCount

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			if peerType == "builder" {

				// ? If all parcels are sent, go to the next block
				if len(parcelsSent) >= len(SplitSamplesIntoParcels(RowCount, ParcelSize)) && blockID < TotalBlocksCount {
					blockID += 1
					parcelsSent = make([]Parcel, 0)
					parcels = SplitSamplesIntoParcels(RowCount, ParcelSize)
				}

				// ? If all blocks are sent, stop
				if blockID >= TotalBlocksCount {
					// log.Println("All blocks sent from builder.")
					continue
				}

				// ? Pick a random parcel that has not been sent yet
				parcelID := 0
				parcelToSend := parcels[parcelID]
				if len(parcels) > 1 {
					parcelID = rand.Intn(len(parcels))
					parcelToSend = parcels[parcelID]
				} else {
					parcels = SplitSamplesIntoParcels(RowCount, ParcelSize)
				}

				parcelSent := false
				for _, p := range parcelsSent {
					// ? If the parcel ID and type are the same, pick another parcel
					if p.StartingIndex == parcelToSend.StartingIndex && p.IsRow == parcelToSend.IsRow {
						parcelSent = true
						break
					}
				}

				// ? If the parcel has been sent, pick another parcel
				if parcelSent {
					continue
				}

				if parcelID != 0 {
					// ? Remove the parcel from the list of parcels to send
					parcels = append(parcels[:parcelID], parcels[parcelID+1:]...)
				}

				// ? Get the samples - 512 bytes per sample
				parcelSamplesToSend := make([]byte, parcelToSend.SampleCount*512)

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

				startTime := time.Now()

				parcelType := "row"
				if !parcelToSend.IsRow {
					parcelType = "col"
				}

				// ? Put parcel samples into DHT
				putErr := dht.PutValue(ctx, "/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(parcelToSend.StartingIndex), parcelSamplesToSend)

				if putErr != nil {
					stats.TotalFailedPuts += 1
					stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))

					log.Print("[BUILDER\t\t" + s.host.ID()[0:5].Pretty() + "] PutValue() Error: " + putErr.Error())
					log.Printf("[BUILDER\t\t"+s.host.ID()[0:5].Pretty()+"]: DHT Peers: %d\n", len(dht.RoutingTable().ListPeers()))

				} else {
					stats.TotalPutMessages += 1
					stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))

					parcelsSent = append(parcelsSent, parcelToSend)

					log.Print(
						printOperation(
							"[BUILDER\t"+s.host.ID()[0:5].Pretty()+"]",
							true,
							true,
							parcelToSend,
							blockID,
							RowCount,
							len(parcelsSent),
							len(SplitSamplesIntoParcels(RowCount, ParcelSize)),
							-1,
						),
					)

				}

			} else if peerType == "validator" {

				if len(parcelsReceived) == 0 && currentBlockID > 0 {
					rowSamplingStartTime = time.Now()
					colSamplingStartTime = time.Now()
					randomSamplingStartTime = time.Now()
				}

				// ? If all parcels are received, go to the next block
				if len(parcelsReceived) >= totalParcelsNeededCount && currentBlockID < TotalBlocksCount {
					currentBlockID += 1
					colParcelsReceivedCount = 0
					rowParcelsReceivedCount = 0
					randomParcelsReceivedCount = 0

					rowSamplingStartTime = time.Now()
					colSamplingStartTime = time.Now()
					randomSamplingStartTime = time.Now()

					rowSamplingLatencyRecorded = false
					colSamplingLatencyRecorded = false
					randomSamplingLatencyRecorded = false

					parcelsReceived = make([]Parcel, 0)
				}

				// ? Get how many row parcels have been received
				if len(parcelsReceived) > 0 {
					for _, p := range parcelsReceived {
						if p.IsRow {
							rowParcelsReceivedCount += 1
						}
					}
				}

				// ? Get how many col parcels have been received
				if len(parcelsReceived) > 0 {
					for _, p := range parcelsReceived {
						if !p.IsRow {
							colParcelsReceivedCount += 1
						}
					}
				}

				// ! Row Parcel Sampling

				// ? Get row parcel
				if rowParcelsReceivedCount < rowParcelsNeededCount {
					// ? Pick a random row parcel that has not been received yet
					parcelID := rand.Intn(len(parcels))
					for _, p := range parcelsReceived {
						// ? If the parcel ID and type are the same, pick another parcel
						if p.StartingIndex == parcels[parcelID].StartingIndex && p.IsRow == parcels[parcelID].IsRow {
							continue
						}
					}

					// ? Get the parcel
					parcelToGet := parcels[parcelID]
					parcelType := "row"
					if !parcelToGet.IsRow {
						parcelType = "col"
					}
					startTime := time.Now()
					_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(currentBlockID)+"/"+parcelType+"/"+fmt.Sprint(parcelToGet.StartingIndex))
					if err != nil {
						stats.TotalFailedGets += 1
						stats.TotalGetMessages += 1
						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
						stats.GetHops = append(stats.GetHops, hops)
					} else {
						stats.TotalGetMessages += 1
						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
						stats.GetHops = append(stats.GetHops, hops)

						parcelsReceived = append(parcelsReceived, parcelToGet)

						log.Print(
							printOperation(
								"[VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
								false,
								true,
								parcelToGet,
								currentBlockID,
								RowCount,
								len(parcelsReceived),
								totalParcelsNeededCount,
								-1,
							),
						)
					}
				} else {
					// log.Print("[VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] Row Sampling Done.")
					if !rowSamplingLatencyRecorded {
						stats.RowSamplingLatencies = append(stats.RowSamplingLatencies, time.Since(rowSamplingStartTime))
						rowSamplingLatencyRecorded = true
					}
				}

				// ! Col Parcel Sampling

				// ? Get col parcel
				if colParcelsReceivedCount < colParcelsNeededCount {
					// ? Pick a random col parcel that has not been received yet
					parcelID := rand.Intn(len(parcels))
					for _, p := range parcelsReceived {
						// ? If the parcel ID and type are the same, pick another parcel
						if p.StartingIndex == parcels[parcelID].StartingIndex && p.IsRow == parcels[parcelID].IsRow {
							continue
						}
					}

					// ? Get the parcel
					parcelToGet := parcels[parcelID]
					parcelType := "row"
					if !parcelToGet.IsRow {
						parcelType = "col"
					}
					startTime := time.Now()
					_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(currentBlockID)+"/"+parcelType+"/"+fmt.Sprint(parcelToGet.StartingIndex))
					if err != nil {
						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
						stats.TotalFailedGets += 1
						stats.TotalGetMessages += 1
						stats.GetHops = append(stats.GetHops, hops)
					} else {
						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
						stats.TotalGetMessages += 1
						stats.GetHops = append(stats.GetHops, hops)

						parcelsReceived = append(parcelsReceived, parcelToGet)

						log.Print(
							printOperation(
								"[VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
								false,
								true,
								parcelToGet,
								currentBlockID,
								RowCount,
								len(parcelsReceived),
								totalParcelsNeededCount,
								-1,
							),
						)
					}
				} else {
					// log.Print("[VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] Col Sampling Done.")
					if !colSamplingLatencyRecorded {
						stats.ColSamplingLatencies = append(stats.ColSamplingLatencies, time.Since(colSamplingStartTime))
						colSamplingLatencyRecorded = true
					}
				}

				// ! 75 Random Parcel Sampling

				if randomParcelsReceivedCount < randomParcelsNeededCount {
					randomSampleID := rand.Intn(TotalSamplesCount)

					// ? Find the parcel that contains the sample
					parcelContainingSample := Parcel{}
					for _, parcel := range parcels {
						parcelSampleIDs := make([]int, parcel.SampleCount)

						if parcel.IsRow {
							for i := 0; i < parcel.SampleCount; i++ {
								parcelSampleIDs[i] = parcel.StartingIndex + i
							}
						} else {
							for i := 0; i < parcel.SampleCount; i++ {
								parcelSampleIDs[i] = parcel.StartingIndex + i*RowCount
							}
						}

						// ? If the parcel contains the sample, break
						found := false
						for _, id := range parcelSampleIDs {
							if id == randomSampleID {
								found = true
								break
							}
						}

						if found {
							parcelContainingSample = parcel
							break
						}

					}

					// ? If the parcel is empty, continue to next loop
					if parcelContainingSample == (Parcel{}) {
						continue
					}

					// ? Check if the parcelContainingSample is in parcelsReceived
					parcelFound := false
					for _, p := range parcelsReceived {
						if p.StartingIndex == parcelContainingSample.StartingIndex && p.IsRow == parcelContainingSample.IsRow {
							parcelFound = true
							break
						}
					}

					// ? If the parcel is already received, continue to next loop
					if parcelFound {
						randomParcelsReceivedCount += 1
						continue
					}

					// ? Get the parcel
					parcelToGet := parcelContainingSample
					parcelType := "row"
					if !parcelToGet.IsRow {
						parcelType = "col"
					}
					startTime := time.Now()
					_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(currentBlockID)+"/"+parcelType+"/"+fmt.Sprint(parcelToGet.StartingIndex))
					if err != nil {
						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
						stats.TotalFailedGets += 1
						stats.TotalGetMessages += 1
						stats.GetHops = append(stats.GetHops, hops)
					} else {
						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
						stats.TotalGetMessages += 1
						stats.GetHops = append(stats.GetHops, hops)

						parcelsReceived = append(parcelsReceived, parcelToGet)
						randomParcelsReceivedCount += 1

						log.Print(
							printOperation(
								"[VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
								false,
								true,
								parcelToGet,
								currentBlockID,
								RowCount,
								len(parcelsReceived),
								totalParcelsNeededCount,
								randomSampleID,
							),
						)
					}
				} else {
					// log.Print("[VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] Random Sampling Done")
					if !randomSamplingLatencyRecorded {
						stats.RandomSamplingLatencies = append(stats.RandomSamplingLatencies, time.Since(randomSamplingStartTime))
						randomSamplingLatencyRecorded = true
					}
				}

			} else if peerType == "nonvalidator" {

				if len(parcelsReceived) == 0 && currentBlockID > 0 {
					randomSamplingStartTime = time.Now()
				}

				// ! 75 Random Parcel Sampling

				// ? If all parcels are received, go to the next block
				if len(parcelsReceived) >= randomParcelsNeededCount && currentBlockID < TotalBlocksCount {
					currentBlockID += 1
					randomParcelsReceivedCount = 0
					randomSamplingStartTime = time.Now()
					randomSamplingLatencyRecorded = false
					parcelsReceived = make([]Parcel, 0)
				}

				if randomParcelsReceivedCount < randomParcelsNeededCount {
					randomSampleID := rand.Intn(TotalSamplesCount)

					// ? Find the parcel that contains the sample
					parcelContainingSample := Parcel{}
					for _, parcel := range parcels {
						parcelSampleIDs := make([]int, parcel.SampleCount)

						if parcel.IsRow {
							for i := 0; i < parcel.SampleCount; i++ {
								parcelSampleIDs[i] = parcel.StartingIndex + i
							}
						} else {
							for i := 0; i < parcel.SampleCount; i++ {
								parcelSampleIDs[i] = parcel.StartingIndex + i*RowCount
							}
						}

						// ? If the parcel contains the sample, break
						found := false
						for _, id := range parcelSampleIDs {
							if id == randomSampleID {
								found = true
								break
							}
						}

						if found {
							parcelContainingSample = parcel
							break
						}

					}

					// ? Check if the parcelContainingSample is in parcelsReceived
					parcelFound := false
					for _, p := range parcelsReceived {
						if p.StartingIndex == parcelContainingSample.StartingIndex && p.IsRow == parcelContainingSample.IsRow {
							parcelFound = true
							break
						}
					}
					// ? If the parcel is already received, continue to next loop
					if parcelFound {
						randomParcelsReceivedCount += 1
						continue
					}

					// ? Get the parcel
					parcelToGet := parcelContainingSample
					parcelType := "row"
					if !parcelToGet.IsRow {
						parcelType = "col"
					}
					startTime := time.Now()
					_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(currentBlockID)+"/"+parcelType+"/"+fmt.Sprint(parcelToGet.StartingIndex))

					if err != nil {
						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
						stats.TotalFailedGets += 1
						stats.TotalGetMessages += 1
						stats.GetHops = append(stats.GetHops, hops)
					} else {
						stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
						stats.TotalGetMessages += 1
						stats.GetHops = append(stats.GetHops, hops)

						parcelsReceived = append(parcelsReceived, parcelToGet)
						randomParcelsReceivedCount += 1

						log.Print(
							printOperation(
								"[NON VALIDATOR\t"+s.host.ID()[0:5].Pretty()+"]",
								false,
								true,
								parcelToGet,
								currentBlockID,
								RowCount,
								len(parcelsReceived),
								randomParcelsNeededCount,
								randomSampleID,
							),
						)
					}
				} else {
					// log.Print("[NON VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] Random Sampling Done")
					if !randomSamplingLatencyRecorded {
						stats.RandomSamplingLatencies = append(stats.RandomSamplingLatencies, time.Since(randomSamplingStartTime))
						randomSamplingLatencyRecorded = true
					}
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
