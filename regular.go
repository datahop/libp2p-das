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

func StartRegularSampling(blockID int, blockDimension int, parcelSize int, s *Service, ctx context.Context, stats *Stats, dht *dht.IpfsDHT){

   startTime := time.Now()

   log.Printf("[R - %s] Starting to sample block %d...\n", s.host.ID()[0:5], blockID)

   randomParcelsNeededCount := 75

   allParcels := SplitSamplesIntoParcels(blockDimension, parcelSize, "all")

   randomParcels := pickRandomParcels(allParcels, randomParcelsNeededCount)

   // Randomize allRandomParcels
   rand.Shuffle(len(randomParcels), func(i, j int) {
      randomParcels[i], randomParcels[j] = randomParcels[j], randomParcels[i]
   })

   log.Printf(
      "[R - %s] Sampling %d random parcels for Block %d...\n",
      s.host.ID()[0:5],
      len(randomParcels),
      blockID,
   )

   sampledParcelIDs := make([]int, 0)
   var parcelWg sync.WaitGroup
   for _, parcel := range randomParcels {
      parcelWg.Add(1)
      go func(p Parcel, blockID int) {
         defer parcelWg.Done()

         parcelType := "col"
         if p.IsRow {
            parcelType = "row"
         }

         for !contains(sampledParcelIDs, p.StartingIndex) {
            //remainingTime := time.Until(startTime.Add(BLOCK_TIME_SEC * time.Second))

            //ctx, cancel := context.WithTimeout(ctx, remainingTime)
            //defer cancel()

            startTime := time.Now()
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

                  if err.Error() == "context deadline exceeded" {
                     break
                  }

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

         }(parcel, blockID)
      }
      parcelWg.Wait()

      stats.TotalSamplingLatencies = append(stats.TotalSamplingLatencies, time.Since(startTime))

      log.Printf("[R - %s] Block %d sampling took %.2f seconds.\n", s.host.ID()[0:5], blockID, time.Since(startTime).Seconds())

}
