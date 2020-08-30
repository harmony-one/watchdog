# watchdog
Harmony Watchdog

Originally from: [harmony-one/harmony-ops](https://github.com/harmony-one/harmony-ops/pull/524)

## How to use
Print a sample configuration file
```bash
./harmony-watchd generate-sample
```

Checks if the IPs listed in the shards are listed in the correct file, according to the reply from the machine. 
Prints errors in JSON format to console.
```bash
./harmony-watchd validate-ip-in-shard
```

Install or remove the daemon from systemd services. (Will require sudo permissions)
```bash
./harmony-watchd service [install] [remove]
```

Starts the monitoring process. Currently does basic sanity checks to validate the config file. 
Also, will return an error if an IP is duplicated in the node-distribution lists.
```bash
./harmony-watchd monitor --yaml-config [config file]
```
## Example YAML file
```yaml
# Place all needed authorization keys here
auth:
  pagerduty:
    event-service-key: YOUR_PAGERDUTY_KEY
  telegram:
    bot-token: YOUR_TELEGRAM_KEY
    chat-id: YOUR_CHATID

network-config:
  target-chain: testnet
  public-rpc: 9500

# How often to check, the numbers assumed as seconds
# block-header RPC must happen first
inspect-schedule:
  block-header: 10
  node-metadata: 15
  cx-pending: 300
  cross-link: 15

# Number of concurrent go threads sending HTTP requests
# Time in seconds to wait for the HTTP request to succeed
performance:
  num-workers: 32
  http-timeout: 1

# Port for the HTML report
http-reporter:
  port: 8080

# Numbers assumed as seconds
shard-health-reporting:
  consensus:
    interval: 10
    warning: 150
  cx-pending:
    pending-limit: 1000
  cross-link:
    warning: 600
  shard-height:
    tolerance: 1000
  connectivity:
    tolerance: 33

# Needs to be an absolute file path
# NOTE: The ending of the basename of the file
# is important, in this example the 0, 1, 2, 3
# indicate shardID. Need to have some trailing
# number on the filename
node-distribution:
  machine-ip-list:
  - /home/ec2-user/mainnet/shard0.txt
  - /home/ec2-user/mainnet/shard1.txt
  - /home/ec2-user/mainnet/shard2.txt
  - /home/ec2-user/mainnet/shard3.txt
```
