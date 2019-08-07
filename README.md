tss
---

Cli and transportation wrapper of [tss-lib](https://github.com/binance-chain/tss-lib)

## Play in localhost

Please note, "--password" option should only be used in testing. 
Without this option, the cli would ask interactive input and confirm

0. build tss executable binary
```
git clone https://github.com/binance-chain/tss
cd tss
go build
```

1. init 3 parties
```
./tss init --home ~/.test1 --moniker "test1" --password "123456789"
./tss init --home ~/.test2 --moniker "test2" --password "123456789"
./tss init --home ~/.test3 --moniker "test3" --password "123456789"
```

2. generate channel id
replace value of "--channel_id" for following commands with generated one
```
./tss channel --channel_expire 30
```

3. keygen 
```
./tss keygen --home ~/.test1 --parties 3 --threshold 1 --password "123456789" --channel_password "123456789" --channel_id "2855D42A535"   
./tss keygen --home ~/.test2 --parties 3 --threshold 1 --password "123456789" --channel_password "123456789" --channel_id "2855D42A535"            
./tss keygen --home ~/.test3 --parties 3 --threshold 1 --password "123456789" --channel_password "123456789" --channel_id "2855D42A535" 

```

4. sign
```
./tss sign --home ~/.test1 --password "123456789" --channel_password "123456789" --channel_id "2855D42A535" 
./tss sign --home ~/.test2 --password "123456789" --channel_password "123456789" --channel_id "2855D42A535" 
```

5. regroup - replace existing 3 parties with 3 brand new parties
```
# init 3 brand new parties
./tss init --home ~/.new1 --moniker "new1" --password "123456789"
./tss init --home ~/.new2 --moniker "new2" --password "123456789"
./tss init --home ~/.new3 --moniker "new3" --password "123456789"

# start 2 old parties (answer Y and n for isOld and IsNew interactive)
./tss regroup --home ~/.test1 --password "123456789" --new_parties 3 --new_threshold 1 --unknown_parties 3 --channel_password "123456789" --channel_id "1565D44EBE1"
./tss regroup --home ~/.test2 --password "123456789" --new_parties 3 --new_threshold 1 --unknown_parties 3 --channel_password "123456789" --channel_id "1565D44EBE1"

# start 3 new parties
./tss regroup --home ~/.new1 --password "123456789" --parties 3 --threshold 1 --new_parties 3 --new_threshold 1 --unknown_parties 4 --channel_password "123456789" --channel_id "1565D44EBE1"
./tss regroup --home ~/.new2 --password "123456789" --parties 3 --threshold 1 --new_parties 3 --new_threshold 1 --unknown_parties 4 --channel_password "123456789" --channel_id "1565D44EBE1"
./tss regroup --home ~/.new3 --password "123456789" --parties 3 --threshold 1 --new_parties 3 --new_threshold 1 --unknown_parties 4 --channel_password "123456789" --channel_id "1565D44EBE1"
```

## Network roles and connection topological
![](network/tss.png)

## Supported NAT Types

Referred to https://github.com/libp2p/go-libp2p/issues/375#issuecomment-407122416 We also have three nat-traversal solutions at the moment.

1. UPnP/NATPortMap 
<br><br> When NAT traversal is enabled (in go-libp2p, pass the NATPortMap() option to the libp2p constructor), libp2p will use UPnP and NATPortMap to ask the NAT's router to open and forward a port for libp2p. If your router supports either UPnP or NATPortMap, this is by far the best option.

2. STUN/hole-punching
<br><br> LibP2P has it's own version of the "STUN" protocol using peer-routing, external address discovery, and reuseport.

3. TURN-like protocol (relay)
<br><br> Finally, we have a TURN like protocol called p2p-circuit. This protocol allows libp2p nodes to "proxy" through other p2p nodes. All party clients registered to mainnet would automatically announce they support p2p-circuit (relay) for tss implementation.



### In WAN setting

| | Full cone | (Address)-restricted-cone | Port-restricted cone	| Symmetric NAT |
| ------ | ------ | ------ | ------ | ------ |
|Bootstrap (tracking) server| ✓ | ✘ | ✘ | ✘ |
|Relay server| ✓ | ✘ | ✘ | ✘ |
|Client| ✓ | ✓ | ✓ | ✓ (relay server needed) |

### In LAN setting

Nodes can connected to each other directly without setting bootstrap and relay server. But host(ip)&port of expectedPeers should be configured.

Refer to `setup_without_bootstrap.sh` to see how LAN direct connection can be used.