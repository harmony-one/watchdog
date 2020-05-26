package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync/atomic"
	"time"
)

func (m *monitor) beaconSyncMonitor(
	beaconBlock, interval, threshold uint64, poolSize int,
	pdServiceKey, chain string, shardMap map[string]int,
) {
	stdlog.Printf("[beaconSyncMonitor] Starting beacon sync check, Beacon Block: %v", beaconBlock)
	currentBeaconHeaders := getBeaconHeaders(poolSize, shardMap)

	shardBeaconMap := map[int]map[uint64]bool{}
	for ip, header := range currentBeaconHeaders {
		if header != nil {
			if beaconBlock > header.Number && beaconBlock-header.Number >= threshold {
				go checkBeaconSync(header.Number, beaconBlock, threshold, interval, ip, pdServiceKey, chain)
			}
			if _, exists := shardBeaconMap[shardMap[ip]]; !exists {
				shardBeaconMap[shardMap[ip]] = map[uint64]bool{}
			}
			if _, exists := shardBeaconMap[shardMap[ip]][header.Number]; !exists {
				shardBeaconMap[shardMap[ip]][header.Number] = true
			}
		}
	}

	for shard, blocks := range shardBeaconMap {
		uniqueBlocks := []uint64{}
		for b := range blocks {
			uniqueBlocks = append(uniqueBlocks, b)
		}
		if shard != 0 {
			sort.SliceStable(uniqueBlocks, func(i, j int) bool {
				return uniqueBlocks[i] > uniqueBlocks[j]
			})
			stdlog.Printf("[beaconSyncMonitor] Shard %d, Beacon height: %d, Unique beacon heights: %v",
				shard, beaconBlock, uniqueBlocks,
			)
		}
	}
}

func getBeaconHeaders(poolSize int,
	shardMap map[string]int,
) map[string]*Header {

	stdlog.Print("[getBeaconHeaders] Fetching latest header data")

	requests := make(chan work)

	go func() {
		defer close(requests)
		requestFields := getRPCRequest(LatestHeadersRPC)
		for n, s := range shardMap {
			if s != 0 {
				requestBody, _ := json.Marshal(requestFields)
				requests <- work{n, LatestHeadersRPC, requestBody}
			}
		}
	}()

	data := make(chan reply)

	workers := int32(poolSize)
	for i := 0; i < poolSize; i++ {
		go func() {
			defer func() {
				if atomic.AddInt32(&workers, -1) == 0 {
					close(data)
				}
			}()

			for r := range requests {
				result := reply{address: r.address, rpc: r.rpc}
				result.rpcResult, result.rpcPayload, result.oops = request("http://"+r.address, r.body)
				data <- result
			}
		}()
	}

	type h struct {
		Result HeaderPair `json:"result"`
	}

	ret := map[string]*Header{}
	success := 0
	for d := range data {
		ret[d.address] = nil

		if d.oops == nil {
			headerReply := h{}
			json.Unmarshal(d.rpcResult, &headerReply)
			ret[d.address] = &headerReply.Result.Beacon
			success++
		}
	}

	stdlog.Printf("[getBeaconHeaders] Successfully fetched %d headers, Failed: %d", success, (len(ret) - success))
	return ret
}

func checkBeaconSync(blockNum, beaconHeight, threshold, syncTimer uint64, IP, pdServiceKey, chain string) {
	type a struct {
		Result NodeMetadataReply `json:"result"`
	}

	stdlog.Printf("[checkBeaconSync] Sleeping %d to check IP %s beacon progress", syncTimer, IP)
	time.Sleep(time.Second * time.Duration(syncTimer))

	type h struct {
		Result HeaderPair `json:"result"`
	}

	requestFields := getRPCRequest(LatestHeadersRPC)
	requestBody, _ := json.Marshal(requestFields)
	result, _, err := request("http://"+IP, requestBody)
	// If error, skip
	if err != nil {
		stdlog.Printf("[checkBeaconSync] Error getting Beacon header: %s", IP)
		return
	}
	headers := h{}
	json.Unmarshal(result, &headers)
	if !(headers.Result.Beacon.Number > blockNum) && (beaconHeight-headers.Result.Beacon.Number > threshold) {
		message := fmt.Sprintf(beaconSyncMessage, IP, headers.Result.Beacon.Number,
			beaconHeight, headers.Result.AuxShard.ShardID, chain,
		)
		incidentKey := fmt.Sprintf("%s beacon out of sync! - %s", IP, chain)
		err := notify(pdServiceKey, incidentKey, chain, message)
		if err != nil {
			errlog.Print(err)
		} else {
		 	stdlog.Printf("[checkBeaconSync] Sent PagerDuty alert! %s", incidentKey)
		}
		stdlog.Printf("[checkBeaconSync] %s beacon not syncing", IP)
	} else {
		stdlog.Printf("[checkBeaconSync] %s beacon sync", IP)
	}
}
