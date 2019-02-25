package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	urpc "github.com/UranusBlockStack/uranus/rpc"
	"github.com/UranusBlockStack/uranus/rpcapi"
	"math/big"
	"os"
)

var (
	rpchost        = "http://localhost:8000"
	cnt     uint64 = 0
)

func sendrawtx(tx *types.Transaction) (utils.Hash, error) {
	// tjson, _ := json.Marshal(tx)
	// fmt.Println("sendrawtx content", string(tjson))

	result := &utils.Hash{}
	bts, _ := rlp.EncodeToBytes(tx)
	signed := "0x" + utils.BytesToHex(bts)

	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		//fmt.Println("sendrawtx diahttp err", err)
		//panic(err)
		return *result, err
	}

	if err := client.Call("Uranus.SendRawTransaction", signed, result); err != nil {
		//fmt.Println("sendrawtx call err", err)
		//panic(err)
		return *result, err
	}

	cnt++
	//fmt.Println(fmt.Sprintf("%6d", cnt), "sendrawtx hash", result.String())
	return *result, err
}

func signandsendtx(data []byte) (utils.Hash, error) {
	result := &utils.Hash{}

	fmt.Println(fmt.Sprintf("%6d", cnt), "signandsendtx content", string(data))
	tx := &rpcapi.SendTxArgs{}
	if err := json.Unmarshal(data, tx); err != nil {
		//fmt.Println("signandsendtx unmarshal err", err)
		//panic(err)
		return *result, err
	}

	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		//fmt.Println("signandsendtx diahttp err", err)
		//panic(err)
		return *result, err
	}

	if err := client.Call("Uranus.SignAndSendTransaction", tx, result); err != nil {
		//fmt.Println("signandsendtx call err", err)
		//panic(err)
		return *result, err
	}

	cnt++
	//fmt.Println(fmt.Sprintf("%6d", cnt), "signandsendtx hash", result.String())
	return *result, err
}

func getnonce(addr utils.Address) uint64 {
	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		fmt.Println("getnonce diahttp err", err)
		panic(err)
	}

	latest := rpcapi.BlockHeight(-1)
	req := &rpcapi.GetNonceArgs{}
	req.Address = addr
	req.BlockHeight = &latest
	result := new(utils.Uint64)
	if err := client.Call("Uranus.GetNonce", req, &result); err != nil {
		fmt.Println("getnonce call err", err)
		panic(err)
	}
	//fmt.Println("getnonce", addr, uint64(*result))
	return uint64(*result)
}

func getGasPrice() big.Int {
	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		fmt.Println("getnonce diahttp err", err)
		panic(err)
	}

	var stringResult string
	if err := client.Call("Uranus.SuggestGasPrice", nil, &stringResult); err != nil {
		fmt.Println("SuggestGasPrice call err", err)
		panic(err)
	}
	result := big.NewInt(0)
	result.SetString(stringResult, 0)

	//fmt.Println("getnonce", addr, uint64(*result))
	return *result
}

func main() {
	signer := types.Signer{}

	rpcHost := flag.String("u", "http://localhost:8000", "RPC host地址")
	issueHex := flag.String("k", "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032", "投票人私钥")
	issuePass := flag.String("p", "", "私钥密码")
	issueAddress := flag.String("a", "", "发送者地址")
	value := flag.Uint64("c", 0, "要投的URAC数量,单位:(urac)")
	boolLogin := flag.Bool("L", false, "注册(-L)还是注销? ")
	boolVote := flag.Bool("V", false, "设置候选还是设置投票(-V)? Candidate or Vote?")
	boolHelp := flag.Bool("h", false, "显示此帮助")

	flag.Parse()
	others := flag.Args()
	rpchost = *rpcHost

	if *boolHelp {

		func() {
			fmt.Fprintf(os.Stderr,
				`
Usage: 
	vote [-h] [-L] [-V] [-u url:port] [<-k pirvite_key>|<-a issue_addr> <-p issue_password>] [-c coin_amount] [address ...]

Options:
`)
			flag.PrintDefaults()

			fmt.Fprintf(os.Stderr,
				`
example:
	vote -L -u 127.0.0.1:8000 -k 953f82bdd854197af9d001e20671cce8ff6274d3012c9c44c6db297faf343b55
	vote -L -V -a 0x970e8128ab834e8eac17ab8e3812f010678cf791 -p "coinbase" address1 address2`)
		}()
		return
	}

	if *boolVote { // 投票操作
		if *boolLogin {
			if len(others) < 1 {
				println("输入的被投票地址数不能少于1个")
				return
			}
			if *value <= 0 {
				println("投票数值不能为0")
				return
			}
		} else {
			if len(others) > 0 {
				println("取消投票不能输入地址")
				return
			}
			if *value > 0 {
				println("取消投票数值必须为0")
				return
			}
		}
	} else { // 候选者操作
		if len(others) > 0 || *value > 0 {
			println("候选者操作不能输入地址和数量")
			return
		}
	}

	var voters []*utils.Address
	for i := 0; i < len(others); i++ {
		if len(others[i]) != 42 {
			println("输入的投票私钥长度:", others[i], " ", len(others[i]), " != 42")
			return
		}

		voters = append(voters, new(utils.Address))
		*voters[i] = utils.HexToAddress(others[i])
		fmt.Println("Voter:", voters[i])
	}

	issueValue := new(big.Int).Mul(big.NewInt(1e18), new(big.Int).SetUint64(*value))
	gasLimit := uint64(21000)
	gasPrice := getGasPrice() //big.NewInt(10000000000)

	if *issuePass != "" && *issueAddress != "" {
		issuer := utils.HexToAddress(*issueAddress)
		fmt.Println("issuer:", issuer)

		var tx string = ""
		if *boolVote {
			if *boolLogin {
				//len(tos)<=30
				var addrs string = ""
				for _, addr := range voters {
					if addrs != "" {
						addrs += ", "
					}
					addrs += string('"') + fmt.Sprint(addr) + string('"')
				}
				tx = fmt.Sprintf(
					`{"TxType":"0x3","From": "%v","Tos": [%v], "Value": "0x%v", "Data": "0x00","Passphrase":"coinbase"}`,
					issuer, addrs, issueValue.Text(16))
			} else {
				// unvote tos == nil
				tx = fmt.Sprintf(
					`{"TxType":"0x4","From": "%v", "Value": "0x0", "Data": "0x00","Passphrase":"coinbase"}`,
					issuer)
			}

		} else {
			if *boolLogin {
				// reg producers tos==nil
				tx = fmt.Sprintf(
					`{"TxType":"0x1","From": "%v", "Value": "0x0", "Data": "0x00","Passphrase":"coinbase"}`,
					issuer)
			} else {
				// unreg tos == nil
				tx = fmt.Sprintf(
					`{"TxType":"0x2","From": "%v", "Value": "0x0", "Data": "0x00","Passphrase":"coinbase"}`,
					issuer)
			}

		}

		result, err := signandsendtx([]byte(tx))
		if err != nil {
			fmt.Println("signandsendtx failed. ", err)
		} else {
			fmt.Println("Result Tx Hash:", result.String())
		}

	} else {
		// issuer
		issuerPriv, _ := crypto.HexToECDSA(*issueHex)
		issuer := crypto.PubkeyToAddress(issuerPriv.PublicKey)
		nonce := getnonce(issuer)
		var txReg *types.Transaction

		if *boolVote {
			if *boolLogin {
				fmt.Println("Transfer(Delegate) from:", issuer.String(), " tos:", voters, " nonce:", nonce)
				txReg = types.NewTransaction(types.Delegate, nonce, issueValue, gasLimit, &gasPrice, nil, voters...)
			} else {
				fmt.Println("Transfer(UnDelegate) from:", issuer.String(), " nonce:", nonce)
				txReg = types.NewTransaction(types.UnDelegate, nonce, issueValue, gasLimit, &gasPrice, nil)
			}
		} else {
			if *boolLogin {
				fmt.Println("Transfer(LoginCandidate) from:", issuer.String(), " nonce:", nonce)
				txReg = types.NewTransaction(types.LoginCandidate, nonce, big.NewInt(0), gasLimit, &gasPrice, nil)
			} else {
				fmt.Println("Transfer(LogoutCandidate) from:", issuer.String(), " nonce:", nonce)
				txReg = types.NewTransaction(types.LogoutCandidate, nonce, big.NewInt(0), gasLimit, &gasPrice, nil)
			}
		}

		txReg.SignTx(signer, issuerPriv)
		result, err := sendrawtx(txReg)
		if err != nil {
			fmt.Println("sendrawtx failed. ", err)
		} else {
			fmt.Println("Result Tx Hash:", result.String())
		}
	}
}
