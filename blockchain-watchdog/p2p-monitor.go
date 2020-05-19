package main

import (
	"fmt"
)

func (m *monitor) p2pMonitor(tolerance int, pdServiceKey, chain string, data MetadataContainer) {
	stdlog.Print("[p2pMonitor] Running p2p connectivity check")
	percent := map[int][]int{}
	for _, metadata := range data.Nodes {
		shard := int(metadata.Payload.ShardID)
		connection := 0
		if metadata.Payload.P2PConnectivity.TotalKnown != 0 {
			connected := float64(metadata.Payload.P2PConnectivity.Connected)
			known := float64(metadata.Payload.P2PConnectivity.TotalKnown)
			connection = int(connected / known * 100)
		}
		percent[shard] = append(percent[shard], connection)
	}
	for shard, values := range(percent) {
		avg := 0
		if len(values) > 0 {
			sum := 0
			for _, v := range values {
				sum = sum + v
			}
			avg = int(float64(sum) / float64(len(values)))
			if avg != 0 && avg < tolerance {
				message := fmt.Sprintf(p2pMessage, shard, avg)
				incidentKey := fmt.Sprintf("Shard %d connectivity lower than threshold - %s", shard, chain)
				err := notify(pdServiceKey, incidentKey, chain, message)
				if err != nil {
					errlog.Print(err)
				} else {
					stdlog.Printf("[p2pMonitor] Send PagerDuty alert! %s", incidentKey)
				}
			}
		}
		stdlog.Printf("[p2pMonitor] Shard: %d, Avg Connectivity: %d%%", shard, avg)
	}
}
