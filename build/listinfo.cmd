@echo off

cls
echo --------------------------------------------------------------
echo @@@@@@[listAccounts]@@@@@@
.\uranuscli.exe -o listAccounts
echo @@@@@@[listPeers]@@@@@@
.\uranuscli.exe -o listPeers
echo @@@@@@[ getConfirmedBlockNumber+getBFTConfirmedBlockNumber]@@@@@@
.\uranuscli.exe -o getLatestBlockHeight
.\uranuscli.exe -o getConfirmedBlockNumber
.\uranuscli.exe -o getBFTConfirmedBlockNumber
echo --------------------------------------------------------------
echo @@@@@@[getBalance on address1,address2,address3,address4]@@@@@@
.\uranuscli.exe -o getBalance 0x970e8128ab834e8eac17ab8e3812f010678cf791
.\uranuscli.exe -o getBalance 0xc08b5542d177ac6686946920409741463a15dddb
.\uranuscli.exe -o getBalance 0xc31c624350733e3aa5b1b70fce4881f93174fd54
.\uranuscli.exe -o getBalance 0x0FE099B53f08bF09Af960362F76926dd317d41B9
echo --------------------------------------------------------------
echo @@@@@@[getCandidates]@@@@@@
.\uranuscli.exe -o getCandidates latest
echo --------------------------------------------------------------
echo @@@@@@[getVoters]@@@@@@
.\uranuscli.exe -o getVoters latest
echo --------------------------------------------------------------
@echo on