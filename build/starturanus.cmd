@echo off
@for %%I in (%0) do @set pth=%%~dpI
@for %%I in (%0) do @set driver=%%~dI
%driver%
cd %pth%
cls
echo -----------------------
cd
echo param[1] = %1
echo -----------------------

if /i not "%2" == "false" ( rd /s /q .\datadir\uranus\chaindata >NUL 2>&1 )
@echo on

@if %1 == 2 goto ura2
@if %1 == 3 goto ura3
@if %1 == 4 goto ura4
@goto ura1

:ura1
echo Start uranus on address1:
.\uranus.exe --datadir datadir -g genesis.json -c config.yaml --node_rpchost 0.0.0.0 --miner_conbase=0x970e8128ab834e8eac17ab8e3812f010678cf791 --log_level info
exit

:ura2
echo Start uranus on address2:
.\uranus.exe --datadir datadir -g genesis.json -c config.yaml --node_rpchost 0.0.0.0 --miner_conbase=0xc08b5542d177ac6686946920409741463a15dddb --log_level info
exit

:ura3
echo Start uranus on address3:
.\uranus.exe --datadir datadir -g genesis.json -c config.yaml --node_rpchost 0.0.0.0 --miner_conbase=0xc31c624350733e3aa5b1b70fce4881f93174fd54 --log_level info
exit

:ura4
echo Start uranus on address4:
.\uranus.exe --datadir datadir -g genesis.json -c config.yaml --node_rpchost 0.0.0.0 --miner_conbase=0x0FE099B53f08bF09Af960362F76926dd317d41B9 --log_level info
exit