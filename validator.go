package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
)

func StartValidatorSampling(blockID int, blockDimension int, parcelSize int, s *Service, ctx context.Context, stats *Stats, dht *dht.IpfsDHT) {

	startTime := time.Now()

	samplesPerRow := blockDimension
	rowColParcelsNeededCount := samplesPerRow / parcelSize

	if samplesPerRow%parcelSize != 0 {
		rowColParcelsNeededCount++
	}

	// randomParcelsNeededCount := 75

	// allParcels := SplitSamplesIntoParcels(blockDimension, parcelSize, "all")
	rowParcels := SplitSamplesIntoParcels(blockDimension, parcelSize, "row")
	colParcels := SplitSamplesIntoParcels(blockDimension, parcelSize, "col")

	randomRowParcels := pickRandomParcels(rowParcels, rowColParcelsNeededCount)
	randomColParcels := pickRandomParcels(colParcels, rowColParcelsNeededCount)
	// randomParcels := pickRandomParcels(allParcels, randomParcelsNeededCount)

	allRandomParcels := append(randomRowParcels, randomColParcels...)
	// allRandomParcels = append(allRandomParcels, randomParcels...)

	// Randomize allRandomParcels
	rand.Shuffle(len(allRandomParcels), func(i, j int) {
		allRandomParcels[i], allRandomParcels[j] = allRandomParcels[j], allRandomParcels[i]
	})

	log.Printf(
		"[V - %s] Sampling %d parcels (%d/%d Rows, %d/%d Cols) for Block %d...\n",
		s.host.ID().String()[0:5],

		len(allRandomParcels),

		len(randomRowParcels),
		rowColParcelsNeededCount,

		len(randomColParcels),
		rowColParcelsNeededCount,

		blockID,
	)

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

			startTime := time.Now()
			for !contains(sampledParcelIDs, p.StartingIndex) {

				returnedPayload, err := dht.GetValue(
					ctx,
					"/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(p.StartingIndex),
				)
				getLatency := time.Since(startTime)
				getTimestamp := time.Now()

				keyHash := sha256.Sum256([]byte("/das/sample/" + fmt.Sprint(blockID) + "/" + parcelType + "/" + fmt.Sprint(p.StartingIndex)))
				keyHashString := fmt.Sprintf("%x", keyHash)

				if err != nil {
					// log.Printf("[V - %s] Failed to get parcel %d: %s\n", s.host.ID()[0:5].Pretty(), p.StartingIndex, err.Error())

					parcelStatus := "fail"
					if err.Error() == "context deadline exceeded" {
						parcelStatus = "timeout"
					}

					stats.GetLatencies = append(stats.GetLatencies, getLatency)
					stats.GetHops = append(stats.GetHops, 0)
					stats.GetTimestamps = append(stats.GetTimestamps, getTimestamp)
					stats.BlockIDs = append(stats.BlockIDs, fmt.Sprint(blockID))
					stats.ParcelKeyHashes = append(stats.ParcelKeyHashes, keyHashString)
					stats.ParcelStatuses = append(stats.ParcelStatuses, parcelStatus)
					stats.ParcelDataLengths = append(stats.ParcelDataLengths, len(returnedPayload))

					stats.TotalFailedGets += 1
					stats.TotalGetMessages += 1
					time.Sleep(1000 * time.Millisecond)

				} else {
					stats.GetLatencies = append(stats.GetLatencies, getLatency)
					stats.GetHops = append(stats.GetHops, 0)
					stats.GetTimestamps = append(stats.GetTimestamps, getTimestamp)
					stats.BlockIDs = append(stats.BlockIDs, fmt.Sprint(blockID))
					stats.ParcelKeyHashes = append(stats.ParcelKeyHashes, keyHashString)
					stats.ParcelStatuses = append(stats.ParcelStatuses, "success")
					stats.ParcelDataLengths = append(stats.ParcelDataLengths, len(returnedPayload))

					stats.TotalGetMessages += 1
					stats.TotalSuccessGets += 1

					sampledParcelIDs = append(sampledParcelIDs, p.StartingIndex)
				}
			}

		}(parcel)
	}
	parcelWg.Wait()
	stats.TotalSamplingLatencies = append(stats.TotalSamplingLatencies, time.Since(startTime))
	log.Printf("[V - %s] Block %d sampling took %.2f seconds.\n", s.host.ID().String()[0:5], blockID, time.Since(startTime).Seconds())
}
