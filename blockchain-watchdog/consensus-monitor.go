package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
)

func (m *monitor) consensusMonitor(
	interval, warning, tolerance uint64, poolSize int,
	pdServiceKey, chain string, shardMap map[string]int,
) {
	jobs := make(chan work, len(shardMap))
	replyChannels := make(map[string](chan reply))
	syncGroups := make(map[string]*sync.WaitGroup)

	replyChannels[BlockHeaderRPC] = make(chan reply, len(shardMap))
	var bhGroup sync.WaitGroup
	syncGroups[BlockHeaderRPC] = &bhGroup

	for i := 0; i < poolSize; i++ {
		go m.worker(jobs, replyChannels, syncGroups)
	}

	requestFields := getRPCRequest(BlockHeaderRPC)

	type s struct {
		Result BlockHeaderReply `json:"result"`
	}

	type lastSuccessfulBlock struct {
		Height uint64
		TS     time.Time
	}

	lastShardData := make(map[string]lastSuccessfulBlock)
	consensusStatus := make(map[string]bool)

	for now := range time.Tick(time.Duration(interval) * time.Second) {
		stdlog.Print("[consensusMonitor] Starting consensus monitor check...")
		queryID := 0
		for n := range shardMap {
			requestFields["id"] = strconv.Itoa(queryID)
			requestBody, _ := json.Marshal(requestFields)
			jobs <- work{n, BlockHeaderRPC, requestBody}
			queryID++
			syncGroups[BlockHeaderRPC].Add(1)
		}
		syncGroups[BlockHeaderRPC].Wait()
		close(replyChannels[BlockHeaderRPC])

		monitorData := BlockHeaderContainer{}
		for d := range replyChannels[BlockHeaderRPC] {
			if d.oops != nil {
				monitorData.Down = append(m.WorkingBlockHeader.Down,
					noReply{d.address, d.oops.Error(),
						string(d.rpcPayload), shardMap[d.address],
					},
				)
			} else {
				oneReport := s{}
				json.Unmarshal(d.rpcResult, &oneReport)
				monitorData.Nodes = append(monitorData.Nodes, BlockHeader{
					oneReport.Result,
					d.address,
				})
			}
		}

		containerCopy := BlockHeaderContainer{}
		containerCopy.Nodes = append([]BlockHeader{}, monitorData.Nodes...)

		go checkShardHeight(containerCopy, warning, tolerance, pdServiceKey, chain)

		blockHeaderData := any{}
		blockHeaderSummary(monitorData.Nodes, true, blockHeaderData)

		currentUTCTime := now.UTC()

		for shard, summary := range blockHeaderData {
			currentBlockHeight := summary.(any)[blockMax].(uint64)
			currentBlockHeader := summary.(any)["latest-block"].(BlockHeader)
			if lastBlock, exists := lastShardData[shard]; exists {
				if currentBlockHeight <= lastBlock.Height {
					timeSinceLastSuccess := currentUTCTime.Sub(lastBlock.TS)
					if timeSinceLastSuccess.Seconds() > 0 && uint64(timeSinceLastSuccess.Seconds()) > warning {
						message := fmt.Sprintf(consensusMessage,
							shard, currentBlockHeight, lastBlock.TS.Format(timeFormat),
							currentBlockHeader.Payload.BlockHash,
							currentBlockHeader.Payload.Leader,
							currentBlockHeader.Payload.ViewID,
							currentBlockHeader.Payload.Epoch,
							currentBlockHeader.Payload.Timestamp,
							currentBlockHeader.Payload.LastCommitSig,
							currentBlockHeader.Payload.LastCommitBitmap,
							int64(timeSinceLastSuccess.Seconds()),
							timeSinceLastSuccess.Minutes(), chain,
						)
						incidentKey := fmt.Sprintf("Shard %s consensus stuck! - %s",
							shard, chain,
						)
						err := notify(pdServiceKey, incidentKey, chain, message)
						if err != nil {
							errlog.Print(err)
						} else {
							stdlog.Print("[consensusMonitor] Sent PagerDuty alert! %s", incidentKey)
						}
						consensusStatus[shard] = false
						continue
					}
				}
			}
			lastShardData[shard] = lastSuccessfulBlock{currentBlockHeight,
				time.Unix(currentBlockHeader.Payload.UnixTime, 0).UTC(),
			}
			consensusStatus[shard] = true
		}
		stdlog.Print(fmt.Sprintf("[consensusMonitor] Total no reply machines: %d", len(monitorData.Down)))
	  for s, b := range consensusStatus {
			stdlog.Print(fmt.Sprintf("[consensusMonitor] Shard %s, Consensus: %v", s, b))
		}

		m.inUse.Lock()
		m.consensusProgress = consensusStatus
		m.inUse.Unlock()
		replyChannels[BlockHeaderRPC] = make(chan reply, len(shardMap))
	}
}

func checkShardHeight(b BlockHeaderContainer, syncTimer, tolerance uint64,
	pdServiceKey, chain string,
) {
	stdlog.Print("[checkShardHeight] Running shard height check...")
	shardHeightMap := make(map[uint32](map[uint64][]BlockHeader))
	for _, v := range b.Nodes {
		shard := v.Payload.ShardID
		block :=  v.Payload.BlockNumber
		if shardHeightMap[shard] == nil {
			shardHeightMap[shard] = make(map[uint64][]BlockHeader)
		}
		if shardHeightMap[shard][block] == nil {
			shardHeightMap[shard][block] = []BlockHeader{}
		}
		shardHeightMap[shard][block] = append(shardHeightMap[shard][block], v)
	}
	for i, s := range shardHeightMap {
		uniqueHeights := []int{}
		for h, _ := range s {
			uniqueHeights = append(uniqueHeights, int(h))
		}

		maxHeight := uint64(0)
		for _, h := range uniqueHeights {
			if uint64(h) > maxHeight {
				maxHeight = uint64(h)
			}
		}
		for _, h := range uniqueHeights {
			if maxHeight - uint64(h) > tolerance {
				for _, v := range shardHeightMap[i][uint64(h)] {
					go checkSync(v.IP, pdServiceKey, chain,
						v.Payload.BlockNumber, maxHeight, syncTimer)
				}
			}
		}
		stdlog.Print(
			fmt.Sprintf("[checkShardHeight] Shard %d, Max height: %d," +
				" Number of unique heights: %d, Unique heights: %v",
				 i, maxHeight, len(uniqueHeights), uniqueHeights),
		)
	}
}

func checkSync(IP, pdServiceKey, chain string,
	blockNumber, shardHeight, syncTimer uint64,
) {
	stdlog.Print(fmt.Sprintf("[checkSync] Sleeping %d to check IP %s progress", syncTimer, IP))
	// Check for progress after checking consensus time
	time.Sleep(time.Second * time.Duration(syncTimer))

	requestFields := getRPCRequest(BlockHeaderRPC)
	requestFields["id"] = strconv.Itoa(0)
	requestBody, _ := json.Marshal(requestFields)
	res := reply{address: IP, rpc: BlockHeaderRPC}
	res.rpcResult, res.rpcPayload, res.oops = request("http://"+IP, requestBody)

	type r struct {
		Result BlockHeaderReply `json:"result"`
	}

	// If invalid reply, no-op
	if res.oops == nil {
		reply := r{}
		json.Unmarshal(res.rpcResult, &reply)
		if !(reply.Result.BlockNumber > blockNumber) {
			message := fmt.Sprintf(blockHeightMessage,
				IP, reply.Result.BlockNumber, shardHeight, reply.Result.ShardID, chain,
			)
			incidentKey := fmt.Sprintf("%s out of sync! - %s", IP, chain)
			err := notify(pdServiceKey, incidentKey, chain, message)
			if err != nil {
				errlog.Print(err)
			} else {
				stdlog.Print(fmt.Sprintf("[checkSync] Sent PagerDuty alert! %s", incidentKey))
			}
			stdlog.Print(fmt.Sprintf("[checkSync] IP %s is not syncing...", IP))
		} else {
			stdlog.Print(fmt.Sprintf("[checkSync] IP %s is syncing...", IP))
		}
	}
}
