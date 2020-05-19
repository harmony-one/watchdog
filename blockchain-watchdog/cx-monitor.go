package main

import (
  "encoding/json"
  "fmt"
  "strconv"
  "sync"
  "time"
)

func (m *monitor) cxMonitor(interval, limit uint64, poolSize int,
  pdServiceKey, chain string, shardMap map[string]int,
) {
	cxRequestFields := getRPCRequest(PendingCXRPC)
	nodeRequestFields := getRPCRequest(NodeMetadataRPC)

	jobs := make(chan work, len(shardMap))
	replyChannels := make(map[string](chan reply))
	syncGroups := make(map[string]*sync.WaitGroup)
	for _, rpc := range []string{NodeMetadataRPC, PendingCXRPC} {
		replyChannels[rpc] = make(chan reply, len(shardMap))
		switch rpc {
		case NodeMetadataRPC:
			var mGroup sync.WaitGroup
			syncGroups[rpc] = &mGroup
		case PendingCXRPC:
			var cxGroup sync.WaitGroup
			syncGroups[rpc] = &cxGroup
		}
	}

	for i := 0; i < poolSize; i++ {
		go m.worker(jobs, replyChannels, syncGroups)
	}

	type r struct {
		Result NodeMetadataReply `json:"result"`
	}

	type a struct {
		Result uint64 `json:"result"`
	}

	for range time.Tick(time.Duration(interval) * time.Second) {
    stdlog.Print("[cxMonitor] Starting cross shard transaction check")
		queryID := 0
		// Send requests to find potential shard leaders
		for n := range shardMap {
			nodeRequestFields["id"] = strconv.Itoa(queryID)
			requestBody, _ := json.Marshal(nodeRequestFields)
			jobs <- work{n, NodeMetadataRPC, requestBody}
			queryID++
			syncGroups[NodeMetadataRPC].Add(1)
		}
		syncGroups[NodeMetadataRPC].Wait()
		close(replyChannels[NodeMetadataRPC])

		leaders := make(map[int][]string)
		for d := range replyChannels[NodeMetadataRPC] {
			if d.oops == nil {
				oneReport := r{}
				json.Unmarshal(d.rpcResult, &oneReport)
				if oneReport.Result.IsLeader {
					shard := int(oneReport.Result.ShardID)
					leaders[shard] = append(leaders[shard], d.address)
				}
			}
		}

		// What do in case of no leader shown (skip cycle for shard)
		// No reply also skip
		queryID = 0
		for _, node := range leaders {
			for _, n := range node {
				cxRequestFields["id"] = strconv.Itoa(queryID)
				requestBody, _ := json.Marshal(cxRequestFields)
				jobs <- work{n, PendingCXRPC, requestBody}
				queryID++
				syncGroups[PendingCXRPC].Add(1)
			}
		}
		syncGroups[PendingCXRPC].Wait()
		close(replyChannels[PendingCXRPC])

		cxPoolSize := make(map[int][]uint64)
		for i := range replyChannels[PendingCXRPC] {
			if i.oops == nil {
				report := a{}
				json.Unmarshal(i.rpcResult, &report)
				shard := 0
				for s, v := range leaders {
					for _, n := range v {
						if n == i.address {
							shard = s
							break
						}
					}
				}
				cxPoolSize[shard] = append(cxPoolSize[shard], report.Result)
				if report.Result > limit {
					message := fmt.Sprintf(crossShardTransactionMessage,
            shard, report.Result,
          )
					incidentKey := fmt.Sprintf(
            "Shard %d cx pool size greater than pending limit! - %s",
            shard, chain,
          )
					err := notify(pdServiceKey, incidentKey, chain, message)
					if err != nil {
						errlog.Print(err)
					} else {
						stdlog.Printf("[cxMonitor] Sent PagerDuty alert: %s", incidentKey)
					}
				}
			}
		}

    for i, v := range cxPoolSize {
      stdlog.Printf("[cxMonitor] Shard: %d, Pending cross shard transaction pool size: %d", i, v)
    }

		replyChannels[NodeMetadataRPC] = make(chan reply, len(shardMap))
		replyChannels[PendingCXRPC] = make(chan reply, len(shardMap))
	}
}
