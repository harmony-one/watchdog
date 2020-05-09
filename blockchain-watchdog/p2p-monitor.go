package main

import (
	"fmt"
)

func (m *monitor) p2pMonitor(tolerance int, pdServiceKey, chain string, data MetadataContainer) {
	stdlog.Print("[p2pMonitor] Running p2p connectivity check...")
	percent := []int{}
	for _, metadata := range data.Nodes {
		if metadata.Payload.P2PConnectivity.TotalKnown != 0 {
			connected := float64(metadata.Payload.P2PConnectivity.Connected)
			known := float64(metadata.Payload.P2PConnectivity.TotalKnown)
			connection := int(connected / known * 100)
			if connection < tolerance {
				message := fmt.Sprintf(p2pMessage, metadata.IP,
					metadata.Payload.P2PConnectivity.Connected,
					metadata.Payload.P2PConnectivity.TotalKnown,
				)
				incidentKey := fmt.Sprintf("Node %s connectivity lower than threshold - %s", metadata.IP, chain)
				err := notify(pdServiceKey, incidentKey, chain, message)
				if err != nil {
					errlog.Print(err)
				} else {
					stdlog.Print("[p2pMonitor] Send PagerDuty alert! %s", incidentKey)
				}
			}
			percent = append(percent, connection)
		}
	}
	if len(percent) > 0 {
		sum := 0
		for _, value := range percent {
			sum = sum + value
		}
		avg := float64(sum) / float64(len(percent))
		stdlog.Print(fmt.Sprintf("[p2pMonitor] Avg Connectivity: %f", avg))
	} else {
		stdlog.Print("[p2pMonitor] Avg Connectivity: N/A")
	}
}
