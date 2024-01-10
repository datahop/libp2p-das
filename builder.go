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

func StartSeedingBlock(blockID int, blockDimension int, parcelSize int, s *Service, ctx context.Context, stats *Stats, dht *dht.IpfsDHT) {

   startTime := time.Now()
   allParcels := SplitSamplesIntoParcels(blockDimension, parcelSize, "all")

   // Randomize allParcels
   rand.Shuffle(len(allParcels), func(i, j int) {
      allParcels[i], allParcels[j] = allParcels[j], allParcels[i]
   })

   log.Printf("[B - %s] Seeding %d parcels for block %d...\n", s.host.ID()[0:5], len(allParcels), blockID)

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
            //remainingTime := time.Until(startTime.Add(BLOCK_TIME_SEC * time.Second))

            //ctx, cancel := context.WithTimeout(ctx, remainingTime)
            //defer cancel()

            putStartTime := time.Now()
            putErr := dht.PutValue(
               ctx,
               "/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(p.StartingIndex),
               parcelSamplesToSend,
            )
            putLatency := time.Since(putStartTime)
            putTimestamp := time.Now()

            keyHash := sha256.Sum256([]byte("/das/sample/" + fmt.Sprint(blockID) + "/" + parcelType + "/" + fmt.Sprint(p.StartingIndex)))
            keyHashString := fmt.Sprintf("%x", keyHash)

            if putErr != nil {
               parcelStatus := "fail"
               if putErr.Error() == "context deadline exceeded" {
                  parcelStatus = "timeout"
               }

               stats.PutLatencies = append(stats.PutLatencies, putLatency)
               stats.PutTimestamps = append(stats.PutTimestamps, putTimestamp)
               stats.BlockIDs = append(stats.BlockIDs, fmt.Sprint(blockID))
               stats.ParcelKeyHashes = append(stats.ParcelKeyHashes, keyHashString)
               stats.ParcelStatuses = append(stats.ParcelStatuses, parcelStatus)

               stats.TotalFailedPuts += 1
               stats.TotalPutMessages += 1

               if putErr.Error() == "context deadline exceeded" {
                  break
               } else if putErr.Error() == "failed to find any peer in table" {
                  break
               } else {
                  log.Printf("[B - %s] Failed to put parcel %d: %s\n", s.host.ID()[0:5], p.StartingIndex, putErr.Error())
               }
            } else {

               // log.Printf("[B - %s] Successfully put parcel %d\n", s.host.ID()[0:5].Pretty(), p.StartingIndex)

               stats.PutLatencies = append(stats.PutLatencies, time.Since(putStartTime))
               stats.PutTimestamps = append(stats.PutTimestamps, putTimestamp)
               stats.BlockIDs = append(stats.BlockIDs, fmt.Sprint(blockID))
               stats.ParcelKeyHashes = append(stats.ParcelKeyHashes, keyHashString)
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

   //log.Printf("[B - %s] Finished seeding block %d in %s (%d/%d)\n", s.host.ID()[0:5], blockID, elapsedTime, stats.TotalSuccessPuts, stats.TotalPutMessages)

   log.Printf("[B - %s] Finished seeding %d parcels.\n", s.host.ID()[0:5], len(allParcels))

}
