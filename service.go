package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
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

func (s *Service) StartMessaging(h host.Host, dht *dht.IpfsDHT, stats *Stats, peerType string, parcelSize int, ctx context.Context) {

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

	if peerType == "builder" {

		for len(dht.RoutingTable().ListPeers()) == 0 {
			log.Printf("[B - %s] Waiting for peers to join...\n", s.host.ID()[0:5].Pretty())
			time.Sleep(time.Second)
		}

		ticker := time.NewTicker(BLOCK_TIME_SEC * time.Second)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			blockID := 0
			for range ticker.C {
				// Add an extra block to give everyone a chance to sample the last block
				if blockID >= TOTAL_BLOCK_COUNT+1 {
					ticker.Stop()
					return
				}

				log.Printf("[B - %s] Starting to seed block %d...\n", s.host.ID()[0:5].Pretty(), blockID)

				startTime := time.Now()

				allParcels := SplitSamplesIntoParcels(ROW_COUNT, parcelSize, "all")

				// Randomize allParcels
				rand.Shuffle(len(allParcels), func(i, j int) {
					allParcels[i], allParcels[j] = allParcels[j], allParcels[i]
				})

				seededParcelIDs := make([]int, 0)

				var parcelWg sync.WaitGroup
				for _, parcel := range allParcels {
					parcelWg.Add(1)
					go func(p Parcel) {
						defer parcelWg.Done()

						parcelSamplesToSend := make([]byte, p.SampleCount*512)

						parcelType := "row"
						if !p.IsRow {
							parcelType = "col"
						}

						for !contains(seededParcelIDs, p.StartingIndex) {
							remainingTime := time.Until(startTime.Add(BLOCK_TIME_SEC * time.Second))

							ctx, cancel := context.WithTimeout(ctx, remainingTime)
							defer cancel()

							putStartTime := time.Now()
							putErr := dht.PutValue(
								ctx,
								"/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(p.StartingIndex),
								parcelSamplesToSend,
							)

							if putErr != nil {
								parcelStatus := "fail"
								if putErr.Error() == "context deadline exceeded" {
									parcelStatus = "timeout"
								}

								stats.PutLatencies = append(stats.PutLatencies, time.Since(putStartTime))
								stats.PutTimestamps = append(stats.PutTimestamps, time.Now().Format("15:04:05.000000"))
								stats.BlockIDs = append(stats.BlockIDs, fmt.Sprint(blockID))
								stats.ParcelIDs = append(stats.ParcelIDs, fmt.Sprint(p.StartingIndex))
								stats.ParcelStatuses = append(stats.ParcelStatuses, parcelStatus)

								stats.TotalFailedPuts += 1
								stats.TotalPutMessages += 1

								if putErr.Error() == "context deadline exceeded" {
									break
								} else {
									log.Printf("[B - %s] Failed to put parcel %d: %s\n", s.host.ID()[0:5].Pretty(), p.StartingIndex, putErr.Error())
								}
							} else {

								// log.Printf("[B - %s] Successfully put parcel %d\n", s.host.ID()[0:5].Pretty(), p.StartingIndex)

								stats.PutLatencies = append(stats.PutLatencies, time.Since(putStartTime))
								stats.PutTimestamps = append(stats.PutTimestamps, time.Now().Format("15:04:05.000000"))
								stats.BlockIDs = append(stats.BlockIDs, fmt.Sprint(blockID))
								stats.ParcelIDs = append(stats.ParcelIDs, fmt.Sprint(p.StartingIndex))
								stats.ParcelStatuses = append(stats.ParcelStatuses, "success")

								stats.TotalSuccessPuts += 1
								stats.TotalPutMessages += 1

								seededParcelIDs = append(seededParcelIDs, p.StartingIndex)
							}
						}
					}(parcel)
				}

				parcelWg.Wait()

				elapsedTime := time.Since(startTime)
				stats.SeedingLatencies = append(stats.SeedingLatencies, elapsedTime)

				log.Printf("[B - %s] Finished seeding block %d in %s\n", s.host.ID()[0:5].Pretty(), blockID, elapsedTime)
				blockID++
			}
		}()
		wg.Wait()

		log.Printf("[B - %s] Finished seeding %d blocks.\n", s.host.ID()[0:5].Pretty(), TOTAL_BLOCK_COUNT)

	} else if peerType == "validator" {

		for blockID := 0; blockID <= TOTAL_BLOCK_COUNT; blockID++ {

			startTime := time.Now()

			log.Printf("[V - %s] Starting to sample block %d...\n", s.host.ID()[0:5].Pretty(), blockID)

			rowColParcelsNeededCount := (ROW_COUNT / 2) / parcelSize
			randomParcelsNeededCount := 75

			allParcels := SplitSamplesIntoParcels(ROW_COUNT, parcelSize, "all")
			rowParcels := SplitSamplesIntoParcels(ROW_COUNT, parcelSize, "row")
			colParcels := SplitSamplesIntoParcels(ROW_COUNT, parcelSize, "col")

			randomRowParcels := pickRandomParcels(rowParcels, rowColParcelsNeededCount)
			randomColParcels := pickRandomParcels(colParcels, rowColParcelsNeededCount)
			randomParcels := pickRandomParcels(allParcels, randomParcelsNeededCount)

			allRandomParcels := append(randomRowParcels, randomColParcels...)
			allRandomParcels = append(allRandomParcels, randomParcels...)

			// Randomize allRandomParcels
			rand.Shuffle(len(allRandomParcels), func(i, j int) {
				allRandomParcels[i], allRandomParcels[j] = allRandomParcels[j], allRandomParcels[i]
			})

			sampledParcelIDs := make([]int, 0)

			var parcelWg sync.WaitGroup
			for _, parcel := range allRandomParcels {
				parcelWg.Add(1)
				go func(p Parcel) {
					defer parcelWg.Done()

					parcelType := "col"
					if p.IsRow {
						parcelType = "row"
					}

					for !contains(sampledParcelIDs, p.StartingIndex) {
						remainingTime := time.Until(startTime.Add(BLOCK_TIME_SEC * time.Second))

						ctx, cancel := context.WithTimeout(ctx, remainingTime)
						defer cancel()

						getStartTime := time.Now()
						_, hops, err := dht.GetValueHops(
							ctx,
							"/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(p.StartingIndex),
						)

						if err != nil {
							// log.Printf("[V - %s] Failed to get parcel %d: %s\n", s.host.ID()[0:5].Pretty(), p.StartingIndex, err.Error())

							parcelStatus := "fail"
							if err.Error() == "context deadline exceeded" {
								parcelStatus = "timeout"
							}

							stats.GetLatencies = append(stats.GetLatencies, time.Since(getStartTime))
							stats.GetHops = append(stats.GetHops, hops)
							stats.GetTimestamps = append(stats.GetTimestamps, time.Now().Format("15:04:05.000000"))
							stats.BlockIDs = append(stats.BlockIDs, fmt.Sprint(blockID))
							stats.ParcelIDs = append(stats.ParcelIDs, fmt.Sprint(p.StartingIndex))
							stats.ParcelStatuses = append(stats.ParcelStatuses, parcelStatus)

							stats.TotalFailedGets += 1
							stats.TotalGetMessages += 1

							if err.Error() == "context deadline exceeded" {
								break
							}
						} else {
							stats.GetLatencies = append(stats.GetLatencies, time.Since(getStartTime))
							stats.GetHops = append(stats.GetHops, hops)
							stats.GetTimestamps = append(stats.GetTimestamps, time.Now().Format("15:04:05.000000"))
							stats.BlockIDs = append(stats.BlockIDs, fmt.Sprint(blockID))
							stats.ParcelIDs = append(stats.ParcelIDs, fmt.Sprint(p.StartingIndex))
							stats.ParcelStatuses = append(stats.ParcelStatuses, "success")

							stats.TotalGetMessages += 1
							stats.TotalSuccessGets += 1

							sampledParcelIDs = append(sampledParcelIDs, p.StartingIndex)

						}
					}

				}(parcel)
			}
			parcelWg.Wait()

			stats.TotalSamplingLatencies = append(stats.TotalSamplingLatencies, time.Since(startTime))

			log.Printf("[V - %s] Block %d sampling took %.2f seconds.\n", s.host.ID()[0:5].Pretty(), blockID, time.Since(startTime).Seconds())

			// Wait until the block time is over
			remainingTime := time.Until(startTime.Add(BLOCK_TIME_SEC * time.Second))
			time.Sleep(remainingTime)

		}

	} else if peerType == "nonvalidator" {

		for blockID := 0; blockID <= TOTAL_BLOCK_COUNT; blockID++ {
			startTime := time.Now()

			log.Printf("[R - %s] Starting to sample block %d...\n", s.host.ID()[0:5].Pretty(), blockID)

			rowColParcelsNeededCount := (ROW_COUNT / 2) / parcelSize
			randomParcelsNeededCount := 75

			allParcels := SplitSamplesIntoParcels(ROW_COUNT, parcelSize, "all")
			rowParcels := SplitSamplesIntoParcels(ROW_COUNT, parcelSize, "row")
			colParcels := SplitSamplesIntoParcels(ROW_COUNT, parcelSize, "col")

			randomRowParcels := pickRandomParcels(rowParcels, rowColParcelsNeededCount)
			randomColParcels := pickRandomParcels(colParcels, rowColParcelsNeededCount)
			randomParcels := pickRandomParcels(allParcels, randomParcelsNeededCount)

			allRandomParcels := append(randomRowParcels, randomColParcels...)
			allRandomParcels = append(allRandomParcels, randomParcels...)

			// Randomize allRandomParcels
			rand.Shuffle(len(allRandomParcels), func(i, j int) {
				allRandomParcels[i], allRandomParcels[j] = allRandomParcels[j], allRandomParcels[i]
			})

			sampledParcelIDs := make([]int, 0)

			var parcelWg sync.WaitGroup
			for _, parcel := range allRandomParcels {
				parcelWg.Add(1)
				go func(p Parcel, blockID int) {
					defer parcelWg.Done()

					parcelType := "col"
					if p.IsRow {
						parcelType = "row"
					}

					for !contains(sampledParcelIDs, p.StartingIndex) {
						remainingTime := time.Until(startTime.Add(BLOCK_TIME_SEC * time.Second))

						ctx, cancel := context.WithTimeout(ctx, remainingTime)
						defer cancel()

						getStartTime := time.Now()
						_, hops, err := dht.GetValueHops(
							ctx,
							"/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(p.StartingIndex),
						)

						if err != nil {
							// log.Printf("[V - %s] Failed to get parcel %d: %s\n", s.host.ID()[0:5].Pretty(), p.StartingIndex, err.Error())

							parcelStatus := "fail"
							if err.Error() == "context deadline exceeded" {
								parcelStatus = "timeout"
							}

							stats.GetLatencies = append(stats.GetLatencies, time.Since(getStartTime))
							stats.GetHops = append(stats.GetHops, hops)
							stats.GetTimestamps = append(stats.GetTimestamps, time.Now().Format("15:04:05.000000"))
							stats.BlockIDs = append(stats.BlockIDs, fmt.Sprint(blockID))
							stats.ParcelIDs = append(stats.ParcelIDs, fmt.Sprint(p.StartingIndex))
							stats.ParcelStatuses = append(stats.ParcelStatuses, parcelStatus)

							stats.TotalFailedGets += 1
							stats.TotalGetMessages += 1

							if err.Error() == "context deadline exceeded" {
								break
							}

						} else {
							stats.GetLatencies = append(stats.GetLatencies, time.Since(getStartTime))
							stats.GetHops = append(stats.GetHops, hops)
							stats.GetTimestamps = append(stats.GetTimestamps, time.Now().Format("15:04:05.000000"))
							stats.BlockIDs = append(stats.BlockIDs, fmt.Sprint(blockID))
							stats.ParcelIDs = append(stats.ParcelIDs, fmt.Sprint(p.StartingIndex))
							stats.ParcelStatuses = append(stats.ParcelStatuses, "success")

							stats.TotalGetMessages += 1
							stats.TotalSuccessGets += 1

							sampledParcelIDs = append(sampledParcelIDs, p.StartingIndex)
						}
					}

				}(parcel, blockID)
			}
			parcelWg.Wait()

			stats.TotalSamplingLatencies = append(stats.TotalSamplingLatencies, time.Since(startTime))

			log.Printf("[R - %s] Block %d sampling took %.2f seconds.\n", s.host.ID()[0:5].Pretty(), blockID, time.Since(startTime).Seconds())

			// Wait until the block time is over
			remainingTime := time.Until(startTime.Add(BLOCK_TIME_SEC * time.Second))
			time.Sleep(remainingTime)

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
