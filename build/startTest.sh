#!/bin/bash

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