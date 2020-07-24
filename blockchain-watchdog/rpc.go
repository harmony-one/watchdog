package main

// RPC definitions
const (
	NodeMetadataRPC   = "hmy_getNodeMetadata"
	BlockHeaderRPC    = "hmy_latestHeader"
	PendingCXRPC      = "hmy_getPendingCXReceipts"
	SuperCommitteeRPC = "hmy_getSuperCommittees"
	LastCrossLinkRPC  = "hmy_getLastCrossLinks"
	LatestHeadersRPC  = "hmy_getLatestChainHeaders"
	JSONVersion       = "2.0"
)

type NodeMetadataReply struct {
	BLSPublicKey   []string `json:"blskey"`
	Version        string   `json:"version"`
	NetworkType    string   `json:"network"`
	IsLeader       bool     `json:"is-leader"`
	ShardID        uint32   `json:"shard-id"`
	NodeRole       string   `json:"role"`
	BlocksPerEpoch int      `json:"blocks-per-epoch"`
	DNSZone        string   `json:"dns-zone,omitempty"`
	ArchivalNode   bool     `json:"is-archival,omitempty"`
	NodeStartTime  int64    `json:"node-unix-start-time"`
	PeerID         string   `json:"peerid"`
	ChainConfig    struct {
		ChainID         int `json:"chain-id"`
		CrossLinkEpoch  int `json:"cross-link-epoch"`
		CrossTxEpoch    int `json:"cross-tx-epoch"`
		Eip155Epoch     int `json:"eip155-epoch"`
		PreStakingEpoch int `json:"prestaking-epoch"`
		S3Epoch         int `json:"s3-epoch"`
		StakingEpoch    int `json:"staking-epoch"`
	} `json:"chain-config"`
	P2PConnectivity struct {
		Connected    int `json:"connected"`
		NotConnected int `json:"not-connected"`
		TotalKnown   int `json:"total-known-peers"`
	} `json:"p2p-connectivity"`
}

type NodeMetadata struct {
	Payload NodeMetadataReply
	IP      string
}

type BlockHeaderReply struct {
	BlockHash        string `json:"blockHash"`
	BlockNumber      uint64 `json:"blockNumber"`
	ShardID          uint32 `json:"shardID"`
	Leader           string `json:"leader"`
	ViewID           uint64 `json:"viewID"`
	Epoch            uint64 `json:"epoch"`
	Timestamp        string `json:"timestamp"`
	UnixTime         int64  `json:"unixtime"`
	LastCommitSig    string `json:"lastCommitSig"`
	LastCommitBitmap string `json:"lastCommitBitmap"`
}

type BlockHeader struct {
	Payload BlockHeaderReply
	IP      string
}

type SuperCommitteeReply struct {
	PreviousCommittee struct {
		Deciders      map[string]CommitteeInfo `json:"quorum-deciders"`
		ExternalCount int                      `json:"external-slot-count"`
	} `json:"previous"`
	CurrentCommittee struct {
		Deciders      map[string]CommitteeInfo `json:"quorum-deciders"`
		ExternalCount int                      `json:"external-slot-count"`
	} `json:"current"`
}

type CommitteeInfo struct {
	PolicyType    string            `json:"policy"`
	MemberCount   int               `json:"count"`
	Externals     int               `json:"external-validator-slot-count"`
	Committee     []CommitteeMember `json:"committee-members"`
	HarmonyPower  string            `json:"hmy-voting-power"`
	StakedPower   string            `json:"staked-voting-power"`
	TotalRawStake string            `json:"total-raw-staked"`
}

type CommitteeMember struct {
	Address        string `json:"earning-account"`
	IsHarmonyNode  bool   `json:"is-harmony-slot"`
	BLSKey         string `json:"bls-public-key"`
	RawPercent     string `json:"voting-power-unnormalized,omitempty"`
	VotingPower    string `json:"voting-power-%"`
	EffectiveStake string `json:"effective-stake,omitempty"`
}

type LastCrossLinkReply struct {
	CrossLinks []CrossLink `json:"result"`
}

type CrossLink struct {
	Hash            string `json:"hash"`
	BlockNumber     int    `json:"block-number"`
	Signature       string `json:"signature"`
	SignatureBitmap string `json:"signature-bitmap"`
	ShardID         int    `json:"shard-id"`
	EpochNumber     int    `json:"epoch-number"`
}

type HeaderPair struct {
	Beacon   Header `json:"beacon-chain-header"`
	AuxShard Header `json:"shard-chain-header"`
}

type Header struct {
	Hash    string `json:"block-header-hash"`
	Number  uint64 `json:"block-number"`
	Epoch   uint64 `json:"epoch"`
	ShardID uint32 `json:"shard-id"`
	ViewID  uint64 `json:"view-id"`
}

func getRPCRequest(rpc string) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": JSONVersion,
		"method":  rpc,
		"params":  []interface{}{},
		"id":      "1",
	}
}
