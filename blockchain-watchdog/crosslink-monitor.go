package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/yanzay/tbot/v2"
)

// Only need to query leader on Shard 0
func (m *monitor) crossLinkMonitor(params watchParams, shardMap map[string]int, tgclient tbot.Client) {
	interval := uint64(params.InspectSchedule.CrossLink)
	warning := uint64(params.ShardHealthReporting.CrossLink.Warning)
	poolSize := int(params.Performance.WorkerPoolSize)
	chain := string(params.Network.TargetChain)
	crossLinkRequestFields := getRPCRequest(LastCrossLinkRPC)
	nodeRequestFields := getRPCRequest(NodeMetadataRPC)

	jobs := make(chan work, len(shardMap))
	replyChannels := make(map[string](chan reply))
	syncGroups := make(map[string]*sync.WaitGroup)

	for _, rpc := range []string{NodeMetadataRPC, LastCrossLinkRPC} {
		replyChannels[rpc] = make(chan reply, len(shardMap))
		switch rpc {
		case NodeMetadataRPC:
			var mGroup sync.WaitGroup
			syncGroups[rpc] = &mGroup
		case LastCrossLinkRPC:
			var lGroup sync.WaitGroup
			syncGroups[rpc] = &lGroup
		}
	}

	for i := 0; i < poolSize; i++ {
		go m.worker(jobs, replyChannels, syncGroups)
	}

	type r struct {
		Result NodeMetadataReply `json:"result"`
	}

	type processedCrossLink struct {
		BlockNum  int
		CrossLink CrossLink
		TS        time.Time
	}

	lastProcessed := make(map[int]processedCrossLink)
	for now := range time.Tick(time.Duration(interval) * time.Second) {
		stdlog.Print("[crossLinkMonitor] Starting crosslink check")
		// Send requests to find potential shard 0 leaders
		for k, v := range shardMap {
			if v == 0 {
				requestBody, _ := json.Marshal(nodeRequestFields)
				jobs <- work{k, NodeMetadataRPC, requestBody}
				syncGroups[NodeMetadataRPC].Add(1)
			}
		}
		syncGroups[NodeMetadataRPC].Wait()
		close(replyChannels[NodeMetadataRPC])

		leader := []string{}
		for d := range replyChannels[NodeMetadataRPC] {
			if d.oops == nil {
				oneReport := r{}
				json.Unmarshal(d.rpcResult, &oneReport)
				if oneReport.Result.IsLeader && oneReport.Result.ShardID == 0 {
					leader = append(leader, d.address)
				}
			}
		}

		// Request from all potential leaders
		for _, l := range leader {
			requestBody, _ := json.Marshal(crossLinkRequestFields)
			jobs <- work{l, LastCrossLinkRPC, requestBody}
			syncGroups[LastCrossLinkRPC].Add(1)
		}
		syncGroups[LastCrossLinkRPC].Wait()
		close(replyChannels[LastCrossLinkRPC])

		crossLinks := LastCrossLinkReply{}
		for i := range replyChannels[LastCrossLinkRPC] {
			if i.oops == nil {
				json.Unmarshal(i.rpcResult, &crossLinks)
				for _, result := range crossLinks.CrossLinks {
					if entry, exists := lastProcessed[result.ShardID]; exists {
						elapsedTime := now.Sub(entry.TS)
						if result.BlockNumber <= entry.BlockNum {
							if uint64(elapsedTime.Seconds()) >= warning {
								message := fmt.Sprintf(crossLinkMessage, result.ShardID,
									result.Hash, result.ShardID, result.BlockNumber, result.ShardID,
									result.EpochNumber, result.Signature, result.SignatureBitmap,
									elapsedTime.Seconds(), elapsedTime.Minutes())
								incidentKey := fmt.Sprintf("Chain: %s, Shard %d, CrossLinkMonitor", chain, result.ShardID)
								errtg := notifytg(tgclient, params.Auth.Telegram.ChatID, incidentKey+"\n"+message)
								err := notify(params.Auth.PagerDuty.EventServiceKey, incidentKey, chain, message)
								if err != nil {
									errlog.Print(err)
								} else {
									stdlog.Printf("[crossLinkMonitor] Sent PagerDuty alert! %s", incidentKey)
								}
								if errtg != nil {
									errlog.Print(errtg)
								} else {
									stdlog.Printf("[crossLinkMonitor] Sent TG alert! %s", incidentKey)
								}
							}
							continue
						}
					}
					lastProcessed[result.ShardID] = processedCrossLink{
						result.BlockNumber,
						result,
						now,
					}
				}
				break
			}
		}
		for s, c := range lastProcessed {
			stdlog.Printf("[crossLinkMonitor] Shard: %d, Last Crosslink: %v", s, c)
		}
		replyChannels[NodeMetadataRPC] = make(chan reply, len(shardMap))
		replyChannels[LastCrossLinkRPC] = make(chan reply, len(shardMap))
		m.inUse.Lock()
		m.LastCrossLinks.CrossLinks = append([]CrossLink{}, crossLinks.CrossLinks...)
		m.inUse.Unlock()
	}
}
