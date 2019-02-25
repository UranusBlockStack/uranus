#!/bin/bash

clear
echo --------------------------------------------------------------
echo @@@@@@[listAccounts]@@@@@@
./uranuscli -o listAccounts
echo @@@@@@[listPeers]@@@@@@
./uranuscli -o listPeers
echo @@@@@@[ getConfirmedBlockNumber+getBFTConfirmedBlockNumber]@@@@@@
./uranuscli -o getLatestBlockHeight
./uranuscli -o getConfirmedBlockNumber
./uranuscli -o getBFTConfirmedBlockNumber
echo --------------------------------------------------------------
echo @@@@@@[getBalance on address1,address2,address3,address4]@@@@@@
./uranuscli -o getBalance 0x970e8128ab834e8eac17ab8e3812f010678cf791
./uranuscli -o getBalance 0xc08b5542d177ac6686946920409741463a15dddb
./uranuscli -o getBalance 0xc31c624350733e3aa5b1b70fce4881f93174fd54
./uranuscli -o getBalance 0x0FE099B53f08bF09Af960362F76926dd317d41B9
echo --------------------------------------------------------------
echo @@@@@@[getCandidates]@@@@@@
./uranuscli -o getCandidates latest
echo --------------------------------------------------------------
echo @@@@@@[getVoters]@@@@@@
./uranuscli -o getVoters latest
echo --------------------------------------------------------------