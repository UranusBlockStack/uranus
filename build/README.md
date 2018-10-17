# build

#### Start uranus

1. public node configuration（params/node.go）
``` 
package params

var BootNodes = []string{
	"enode://1fec388652dc57e59fd5d54555036b5267f575c3bac0ddf6cb725db5a3f4d305f715fe0612944dcd640461ae47bfa7cc4ffc6a42ee5179025960be3ee8536fea@127.0.0.1:7090",
}

```

- encode://publickey@ip:port, now the public node uses the local `127.0.0.1`，private key file use `./nodekey`。
- Program startup will find the file nodekey from the current directory(`./nodekey`) if not found, generate the private key, stored in the data directory nodekey.
  ```
  INFO[2018-08-29 09:36:20] Starting P2P networking                       caller="server.go:89"
  INFO[2018-08-29 09:36:20] UDP listener up self enode://1fec388652dc57e59fd5d54555036b5267f575c3bac0ddf6cb725db5a3f4d305f715fe0612944dcd640461ae47bfa7cc4ffc6a42ee5179025960be3ee8536fea@[::]:7090  caller="udp.go:237" 
  ```

2. configuration
- datadir: data dir
- miner_start: start miner,default does not start mining 
- p2p_listenaddr: p2p listen url, default`127.0.0.1:7090`
- node_rpcport: rpc listen port, default`:8000`

#### Local multi process node test
1. public node is available
2. execue script
 ```

ps x | grep uranus | awk '{print $1}' | xargs kill >null 2>&1
rm -f null

cd ..
./build/uranus  --miner_start > uranus.log 2>&1 &

cd build
# start uranus
for i in 1 2 3 4 5
do
	mkdir -p datadir$i
	./uranus --miner_start --datadir datadir$i --p2p_listenaddr :707$i --node_rpcport 800$i > datadir$i/uranus.log 2>&1 &
done

 ```

 #### Use vagrant test multi nodes（need install vagrant）
 1. change code ip，modify public node ip(127.0.0.1) is avaliable ip. 
 2. make & make run 
 2. cd devenv & vagrant destroy & vagrant up 
 4. vagrant ssh node1 (log:`~/uranus.log`)
 ```
 If you want to start vagrant faster, change `setup.sh`. annotate the first line, and cancel the second line and set `go1.9.2.linux-amd64.tar` in current directory.
 
 ```