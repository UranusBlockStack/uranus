package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	urpc "github.com/UranusBlockStack/uranus/rpc"
	"github.com/UranusBlockStack/uranus/rpcapi"
)

var (
	rpchost = "http://localhost:8000"

	delay            uint = 10
	delayGrad        uint = 10
	delayAlteredTime      = time.Now()
	delayOptimum     uint = 0
	delayOptimumTime      = time.Now()
	useAutoDelay     bool = true

	useTxVoter      bool
	useTxCandidate  bool
	useTxTranster   bool
	useLongTimeTest bool
)

var cnt uint64 = 0
var errcnt uint64 = 0
var boolPntDetail = false

func pause() {
	fmt.Println("请按任意键继续...")
	var c rune
	fmt.Scanf("%c", &c)
}

func sendrawtransaction(client *urpc.Client, tx *types.Transaction, nonce uint64) (utils.Hash, error) {
	// tjson, _ := json.Marshal(tx)
	// fmt.Println("sendrawtransaction content", string(tjson))

	result := &utils.Hash{}
	bts, _ := rlp.EncodeToBytes(tx)
	signed := "0x" + utils.BytesToHex(bts)
	err := client.Call("Uranus.SendRawTransaction", signed, result)
	if err != nil {
		errcnt++
	}

	if boolPntDetail {
		if err == nil {
			fmt.Printf("Tx[%6d][%6d]:%s\r\n", cnt, nonce, result.String())
		} else {
			fmt.Printf("Tx[%6d][%6d]:FAILURE!\r\n", cnt, nonce)
		}

	} else {
		if cnt%10 == 0 {
			fmt.Printf("\r\nTx[%6d]: ", cnt)
		}

		if err == nil {
			fmt.Printf("[%6d] ", nonce)
		} else if strings.Contains(err.Error(), "transaction underpriced") {
			fmt.Printf("<%6d> ", nonce)
		} else if strings.Contains(err.Error(), "nonce") {
			fmt.Printf("!%6d! ", nonce)
		} else if strings.Contains(err.Error(), "known transaction") {
			fmt.Printf("!%6d! ", nonce)
		} else {
			fmt.Printf(" X%6dX ", nonce, err, " ")
		}

	}
	cnt++

	return *result, err
}

func getnonce(client *urpc.Client, addr utils.Address) uint64 {
	latest := rpcapi.BlockHeight(-1)
	req := &rpcapi.GetNonceArgs{}
	req.Address = addr
	req.BlockHeight = &latest
	result := new(utils.Uint64)
	if err := client.Call("Uranus.GetNonce", req, &result); err != nil {
		fmt.Println("getnonce call err", err)
		if !useLongTimeTest {
			panic(err)
			return 0
		} else {
			time.Sleep(60 * time.Second)
			return getnonce(client, addr)
		}
	}
	//fmt.Println("getnonce", addr, uint64(*result))
	return uint64(*result)
}

func txPoolSize(client *urpc.Client) uint64 {
	result := map[string]utils.Uint{}
	if err := client.Call("TxPool.Status", "", &result); err != nil {
		panic(err)
	}
	size := uint64(0)
	for _, s := range result {
		size += uint64(s)
	}
	return size
}

func txQueuedSize(client *urpc.Client) uint64 {
	result := map[string]utils.Uint{}
	if err := client.Call("TxPool.Status", "", &result); err != nil {
		panic(err)
	}
	return uint64(result["queued"])
}

func getbalance(client *urpc.Client, addr utils.Address) utils.Big {
	result := new(utils.Big)
	req := &rpcapi.GetBalanceArgs{}
	req.Address = addr
	if err := client.Call("Uranus.GetBalance", req, &result); err != nil {
		fmt.Println("getbalance call err", err)
		if !useLongTimeTest {
			panic(err)
			return *result
		} else {
			time.Sleep(60 * time.Second)
			return getbalance(client, addr)
		}
	}

	return *result
}

func checkTxByHash(client *urpc.Client, hash utils.Hash) int {
	result := &rpcapi.RPCTransaction{}
	if err := client.Call("BlockChain.GetTransactionByHash", hash, &result); err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Print("$")
		} else {
			fmt.Println("checkTxByHash call err:", err)
		}

		return 0
	}

	if result.Hash != hash {
		fmt.Println("checkTxByHash recive a wrong hash.")
		return 0
	}

	if result.BlockHeight != nil && result.BlockHeight.ToInt().Cmp(big.NewInt(0)) > 0 {
		//fmt.Println("checkTxByHash checked Tx in Block with Height[", result.BlockHeight.ToInt().String(), "]")
		return 2 // 即使交易已经打包入区块也不能确定交易是确认的,因为打包的区块还有可能被回滚
	}

	if result.BlockHeight != nil {
		fmt.Println("checkTxByHash unchecked Block Height******:", result.BlockHeight.ToInt().String())
	}

	return 1
}

func main() {
	flag.StringVar(&rpchost, "u", "http://localhost:8000", "RPC host地址")
	intWait := flag.Int("w", 12, "出错时停等时间")

	flag.BoolVar(&useLongTimeTest, "l", false, "是否进行长期稳定性测试")
	flag.BoolVar(&boolPntDetail, "d", false, "是否打印详细信息")

	intThreadCnt := flag.Int("tread", 1, "测试用线程数,0表示CPU个数")
	intAddressCnt := flag.Int("addr", 1, "每个线程操作几个地址进行测试")
	noAutoNonceCheck := flag.Bool("no-auto-nonce", false, "是否开启校准nonce模式")
	noAutoDelay := flag.Bool("no-auto-delay", false, "是否使用自动延迟功能")
	intTxDelay := flag.Int("delay", 10, "每笔交易延迟,只有开启no-auto-delay才生效(单位:ms)")

	flag.BoolVar(&useTxTranster, "T", false, "是否使用代币转账事务")
	flag.BoolVar(&useTxVoter, "V", false, "是否使用投票事务")
	flag.BoolVar(&useTxCandidate, "C", false, "是否使用候选事务")
	flag.Parse()

	if *noAutoDelay {
		useAutoDelay = false
		if *intTxDelay >= 0 {
			delay = (uint)(*intTxDelay)
		}else{
			fmt.Println("交易延迟不能为负值.")
			delay = 10
		}
	}else{
		useAutoDelay = true
		delay = 100
	}

	boolCheckNonce := true
	if *noAutoNonceCheck {boolCheckNonce = false}

	if !useTxTranster && !useTxVoter && !useTxCandidate {
		useTxTranster = true
		useTxVoter = true
		useTxCandidate = true
	} else if useTxVoter {
		useTxCandidate = true
	}

	if len(rpchost) < 7 {
		rpchost = "http://localhost:8000"
	}
	if !strings.Contains(strings.ToLower(rpchost), "http") {
		rpchost = "http://" + rpchost
	}
	if !strings.ContainsRune(string([]rune(rpchost)[len(rpchost)-7:]), ':') {
		rpchost = rpchost + ":8000"
	}

	if *intThreadCnt == 0 {
		*intThreadCnt = runtime.NumCPU()
	}

	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		panic(fmt.Sprintf("diahttp err: %v", err))
	}

	// 1. generate addresses
	// 2. transfer addresses
	// 3. worker
	// 4. transfer reg vote unvote unreg
	signer := types.Signer{}
	issuePrivHex := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	issuerNonce := uint64(0)
	issueValue := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100000))
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(10000000000)
	tpsSize := *intThreadCnt * *intAddressCnt
	wokers := *intThreadCnt

	if tpsSize < wokers {
		tpsSize = wokers
	}

	// issuer
	issuerPriv, _ := crypto.HexToECDSA(issuePrivHex)
	issuer := crypto.PubkeyToAddress(issuerPriv.PublicKey)
	fmt.Println("issuer", issuer)
	issuerNonce = getnonce(client, issuer)

	//
	var timeStart, timeEnd time.Time

	// generate addresses
	tps := []*ecdsa.PrivateKey{}
	tpsOver := []*ecdsa.PrivateKey{}
	for i := 0; i < tpsSize; i++ {
		priv, _ := crypto.GenerateKey()
		tps = append(tps, priv)
	}

	// transfer addresses
	for _, priv := range tps {
		addr := crypto.PubkeyToAddress(priv.PublicKey)
		txTransfer := types.NewTransaction(types.Binary, issuerNonce, issueValue, gasLimit, gasPrice, nil, []*utils.Address{&addr}...)
		txTransfer.SignTx(signer, issuerPriv)
		_, err := sendrawtransaction(client, txTransfer, issuerNonce)
		if err != nil {
			fmt.Println("sendrawtransaction fail.", err)
			value := getbalance(client, issuer)
			fmt.Println("ADDR:", issuer.String(), "Balance:", value.ToInt().String())
			fmt.Println("Need:", issueValue.String(), "*", tpsSize)
			return
		}

		issuerNonce++
	}

	time.Sleep(6 * time.Second)
	fmt.Println("\r\n==================================================================")
	for _, priv := range tps {
		addr := crypto.PubkeyToAddress(priv.PublicKey)
		value := getbalance(client, addr)
		gotnonce := getnonce(client, addr)
		fmt.Println("ADDR:", addr.String(), "Nonce:", gotnonce, "Value:", value.String())
	}

	fmt.Println("\r\n")
	fmt.Print("Tx Transfer:[", useTxTranster, "] Tx Candidate:[", useTxCandidate, "] Tx Vote:[", useTxVoter, "]\r\n")
	fmt.Println("==================================================================")
	pause()

	timeStart = time.Now()
	// workes
	wg := &sync.WaitGroup{}
	wg.Add(wokers)
	cnt := len(tps) / wokers
	for i := 0; i < wokers; i++ {
		f := i * cnt
		t := (i + 1) * cnt
		ttps := tps[f:t]

		go func(tps []*ecdsa.PrivateKey) {
			defer wg.Done()
			client, err := urpc.DialHTTP(rpchost)
			if err != nil {
				fmt.Printf("diahttp err: %v", err)
				return
			}
			nonces := map[utils.Address]uint64{}
			nonceconfirms := map[utils.Address]uint64{}
			nonceconfirmtime := map[utils.Address]time.Time{}

			txhashs := map[uint64]utils.Hash{}

			for _, priv := range tps {
				selAddr := crypto.PubkeyToAddress(priv.PublicKey)
				nonces[selAddr] = getnonce(client, selAddr)
			}

			tpscount := len(tps)
			for tpscount > 0 {
				for index, priv := range tps {
					if priv == nil {
						continue
					}

					selAddr := crypto.PubkeyToAddress(priv.PublicKey)
					tpriv, _ := crypto.GenerateKey()
					addr := crypto.PubkeyToAddress(tpriv.PublicKey)
					nonce := nonces[selAddr]
					lastConfirm := nonceconfirms[selAddr]
					lastConfirmTime := nonceconfirmtime[selAddr]
					if lastConfirmTime.IsZero() {
						lastConfirmTime = time.Now()
					}

					funcParseResult := func(hs utils.Hash, inErr error) (res bool) {
						if inErr != nil && strings.Contains(inErr.Error(), "insufficient funds for gas * price + value") {
							tpsOver = append(tpsOver, tps[index])
							tps[index] = nil
							tpscount--
							return false
						}

						if inErr == nil {
							nonce++
							time.Sleep(time.Duration(delay) * time.Millisecond)
							if useAutoDelay && time.Now().Sub(delayAlteredTime).Seconds() > 30 {
								delayAlteredTime = time.Now()

								if ts := txPoolSize(client); ts > 30000 {
									delay += delayGrad
									fmt.Print("\r\nCurrent delay: ", delay, " ^+", delayGrad, "\r\n")
								} else if delayOptimum > 0 && delay <= delayOptimum {
									if delay > 0 {
										delay--
									}
									fmt.Print("\r\nCurrent delay: ", delay, " ^-1 Optimun:", delayOptimum, "\r\n")
								} else if delay >= delayGrad {
									delay -= delayGrad
									fmt.Print("\r\nCurrent delay: ", delay, " ^-", delayGrad, "\r\n")
								} else {
									fmt.Print("\r\nCurrent delay: 0 ^-", delay, "\r\n")
									delay = 0
								}
							}

							if !boolCheckNonce {
								return true
							}
						}

						if boolCheckNonce {

							if nonce < lastConfirm {
								fmt.Println("\n\n*******************************************************")
								fmt.Println("nonce:", nonce, "< lastConfirm:", lastConfirm)
								fmt.Println("*******************************************************")
								if !useLongTimeTest {
									panic("nonce < lastConfirm")
								}

								if nonce > 0 {
									lastConfirm = nonce
								}
							}

							if inErr == nil {
								txhashs[nonce-1] = hs
							}

							if inErr != nil || nonce-lastConfirm > 100 || time.Now().Sub(lastConfirmTime).Seconds() > 10 {
								getnc := getnonce(client, selAddr)

								if getnc > nonce {
									for i := lastConfirm; i < nonce; i++ {
										delete(txhashs, i)
									}
									nonce = getnc
									lastConfirm = getnc
									lastConfirmTime = time.Now()

								} else if getnc > lastConfirm {
									for i := lastConfirm; i < getnc; i++ {
										delete(txhashs, i)
									}

									lastConfirm = getnc
									lastConfirmTime = time.Now()

								} else if getnc < nonce { // getnc == lastConfirm
									if getnc < lastConfirm {
										fmt.Println("***************************************************")
										fmt.Println("getnc:", getnc, "lastConfirm:", lastConfirm)
										fmt.Println("***************************************************")
										if !useLongTimeTest {
											panic("ERROR: getnc < lastConfirm")
										}
										lastConfirm = getnc
									}

									for i := 0; i < 10; i++ {
										size := txQueuedSize(client)
										if size == 0 {
											break
										}
										fmt.Println("########queued size#########", size)
										time.Sleep(500 * time.Millisecond)
									}

									var bCheckFailure uint64 = 0
									var firstFailure uint64 = 0
									oldnonce := nonce

									for i := lastConfirm; ; i++ {
										hash, ok := txhashs[i]
										if !ok {
											nonce = i
											break
										}
										chk := checkTxByHash(client, hash)
										if chk <= 0 {
											if bCheckFailure == 0 {
												firstFailure = i
												fmt.Println("\r\nTxs Execute Failure!")

												if useAutoDelay {
													if (delayOptimum == 0 || delayOptimum > delay+delayGrad || time.Now().Sub(delayOptimumTime).Seconds() > 600) &&
														delay < 100 {
														delayOptimumTime = time.Now()
														delayOptimum = delay + delayGrad
													}

													if time.Now().Sub(delayAlteredTime).Seconds() < 1 {
														delay += delayGrad * 10
														fmt.Print("\r\nCurrent delay: ", delay, " ^+", delayGrad*10, "\r\n")
													} else if time.Now().Sub(delayAlteredTime).Seconds() < 5 {
														delay += delayGrad * 5
														fmt.Print("\r\nCurrent delay: ", delay, " ^+", delayGrad*5, "\r\n")
													} else {
														delay += delayGrad
														fmt.Print("\r\nCurrent delay: ", delay, " ^+", delayGrad, "\r\n")
													}
												}

												if *intWait > 0 {
													fmt.Println("Tx failure. time wait", time.Duration(*intWait)*time.Second)
													time.Sleep(time.Duration(*intWait) * time.Second)
												}
											}

											bCheckFailure++

											val := new(big.Int).Mul(big.NewInt(1e10), big.NewInt(1))
											txTransfer := types.NewTransaction(types.Binary, i, val, gasLimit, gasPrice, nil, []*utils.Address{&addr}...)
											txTransfer.SignTx(signer, priv)
											newhs, err := sendrawtransaction(client, txTransfer, i)
											if err == nil {
												txhashs[i] = newhs
											}

											time.Sleep(time.Duration(delay) * time.Millisecond)

										} else if chk == 1 {
											continue
										} else { // == 2
											//lastConfirm = i + 1
											//lastConfirmTime = time.Now()
											//delete(txhashs, i)
										}
									}

									if bCheckFailure > 0 && useAutoDelay {
										right := nonce - firstFailure - bCheckFailure
										fmt.Println("\r\nTxs Execute Failure!", " CONFIRM:", lastConfirm, " First FAIL:", firstFailure,
											" FAIL COUNT:", bCheckFailure, " RIGHT COUNT:", right, " LOCAL NONCE:", oldnonce, "->", nonce)

										if *intWait > 0 {
											fmt.Println("Tx failure. time wait", time.Duration(*intWait)*time.Second)
											time.Sleep(time.Duration(*intWait) * time.Second)
										}

										delayAlteredTime = time.Now()
									}
								}
							}
						}

						if inErr == nil {
							return true
						}

						if !boolCheckNonce && useAutoDelay {
							delay += delayGrad
							delayAlteredTime = time.Now()
							fmt.Print("\r\nCurrent delay: ", delay, " ^+", delayGrad, "\r\n")
						}

						if strings.Contains(inErr.Error(), "transaction underpriced") {
							fmt.Println("Tx ERROR:", inErr)

						} else if strings.Contains(inErr.Error(), "nonce") ||
							strings.Contains(inErr.Error(), "known transaction") {

							if !boolCheckNonce {
								oldnonce := nonce
								nonce = getnonce(client, selAddr)
								fmt.Println("\r\n!Nonce Changed! LOCAL:", oldnonce, "->", nonce)
							}

							fmt.Println("Tx ERROR:", inErr)

						} else {
							fmt.Println("\r\nSendTx Unknown Mistake. ERR:", inErr)
						}

						if *intWait > 0 {
							fmt.Println("Tx failure. time wait:", time.Duration(*intWait)*time.Second)
							time.Sleep(time.Duration(*intWait) * time.Second)
						}

						return false
					}

					// reg
					if useTxCandidate {
						txReg := types.NewTransaction(types.LoginCandidate, nonce, big.NewInt(0), gasLimit, gasPrice, nil)
						txReg.SignTx(signer, priv)
						if !funcParseResult(sendrawtransaction(client, txReg, nonce)) {
							nonces[selAddr] = nonce
							nonceconfirms[selAddr] = lastConfirm
							nonceconfirmtime[selAddr] = lastConfirmTime
							continue
						}
					}

					// transfer
					if useTxTranster {
						val := new(big.Int).Mul(big.NewInt(1e10), big.NewInt(1))
						txTransfer := types.NewTransaction(types.Binary, nonce, val, gasLimit, gasPrice, nil, []*utils.Address{&addr}...)
						txTransfer.SignTx(signer, priv)
						if !funcParseResult(sendrawtransaction(client, txTransfer, nonce)) {
							nonces[selAddr] = nonce
							nonceconfirms[selAddr] = lastConfirm
							nonceconfirmtime[selAddr] = lastConfirmTime
							continue
						}
					}

					// vote
					if useTxVoter {
						valVote := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1))
						txVote := types.NewTransaction(types.Delegate, nonce, valVote, gasLimit, gasPrice, nil, []*utils.Address{&selAddr}...)
						txVote.SignTx(signer, priv)
						if !funcParseResult(sendrawtransaction(client, txVote, nonce)) {
							nonces[selAddr] = nonce
							nonceconfirms[selAddr] = lastConfirm
							nonceconfirmtime[selAddr] = lastConfirmTime
							continue
						}
					}

					// unvote
					if useTxVoter {
						txUnvote := types.NewTransaction(types.UnDelegate, nonce, big.NewInt(0), gasLimit, gasPrice, nil)
						txUnvote.SignTx(signer, priv)
						if !funcParseResult(sendrawtransaction(client, txUnvote, nonce)) {
							nonces[selAddr] = nonce
							nonceconfirms[selAddr] = lastConfirm
							nonceconfirmtime[selAddr] = lastConfirmTime
							continue
						}
					}

					// unreg
					if useTxCandidate {
						txUnReg := types.NewTransaction(types.LogoutCandidate, nonce, big.NewInt(0), gasLimit, gasPrice, nil)
						txUnReg.SignTx(signer, priv)
						if !funcParseResult(sendrawtransaction(client, txUnReg, nonce)) {
							nonces[selAddr] = nonce
							nonceconfirms[selAddr] = lastConfirm
							nonceconfirmtime[selAddr] = lastConfirmTime
							continue
						}
					}

					nonces[selAddr] = nonce
					nonceconfirms[selAddr] = lastConfirm
					nonceconfirmtime[selAddr] = lastConfirmTime
				}
			}
		}(ttps)
	}

	wg.Wait()
	timeEnd = time.Now()

	fmt.Printf("\n\n\n")
	fmt.Println("==================================================================")
	for _, priv := range tpsOver {
		addr := crypto.PubkeyToAddress(priv.PublicKey)
		value := getbalance(client, addr)
		fmt.Println("ADDR:", addr.String(), "Value:", value.String())
	}

	fmt.Println("==================================================================")
	fmt.Println("Address Count:", tpsSize)
	fmt.Println("Thread Count :", wokers)
	fmt.Println("Time Elapse  :", timeEnd.Sub(timeStart).String())
	fmt.Println("Success Tx count:", cnt)
	fmt.Println("Failure Tx count:", errcnt)
}
