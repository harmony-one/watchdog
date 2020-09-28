package main

import (
	"fmt"

	"github.com/yanzay/tbot/v2"
)

func (m *monitor) p2pMonitor(params watchParams, data MetadataContainer, tgclient tbot.Client) {
	stdlog.Print("[p2pMonitor] Running p2p connectivity check")
	percent := map[int][]int{}
	chain := string(params.Network.TargetChain)
	tolerance := int(params.ShardHealthReporting.Connectivity.Warning)

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
	for shard, values := range percent {
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
				errtg := notifytg(tgclient, params.Auth.Telegram.ChatID, incidentKey+"\n"+message)
				err := notify(params.Auth.PagerDuty.EventServiceKey, incidentKey, chain, message)
				if err != nil {
					errlog.Print(err)
				} else {
					stdlog.Printf("[p2pMonitor] Send TG/PagerDuty alert! %s", incidentKey)
				}
				if errtg != nil {
					errlog.Print(errtg)
				} else {
					stdlog.Printf("[p2pMonitor] Sent TG alert! %s", incidentKey)
				}
			}
		}
		stdlog.Printf("[p2pMonitor] Shard: %d, Avg Connectivity: %d%%", shard, avg)
	}
}
