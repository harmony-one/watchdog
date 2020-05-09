package main

const (
	consensusMessage = `
Consensus stuck on shard %s!

Block height stuck at %d starting at %s

Block Hash: %s

Leader: %s

ViewID: %d

Epoch: %d

Block Timestamp: %s

LastCommitSig: %s

LastCommitBitmap: %s

Time since last new block: %d seconds (%f minutes)

See: http://watchdog.hmny.io/report-%s
`
	crossShardTransactionMessage = `
Cx Transaction Pool too large on shard %d!

Count: %d
`
	crossLinkMessage = `
Haven't processed a cross link for shard %d in a while!

Cross Link Hash: %s

Shard %d Block: %d

Shard %d Epoch: %d

Signature: %s

Signature Bitmap: %s

Time since last processed cross link: %f seconds (%f minutes)
`
	blockHeightMessage = `
%s at block height %d, which shard height %d.

Shard: %d

Chain: %s
`
	p2pMessage = `
Shard: %s

Avg Connectivity: %d
`
)
