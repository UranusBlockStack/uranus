package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"
	"strings"
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
)

var cnt uint64 = 0

func sendrawtransaction(tx *types.Transaction) (utils.Hash, error) {
	// tjson, _ := json.Marshal(tx)
	// fmt.Println("sendrawtransaction content", string(tjson))

	result := &utils.Hash{}
	bts, _ := rlp.EncodeToBytes(tx)
	signed := "0x" + utils.BytesToHex(bts)

	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		fmt.Println("sendrawtransaction diahttp err", err)
		//panic(err)
		return *result, err
	}
	defer client.Close()

	if err := client.Call("Uranus.SendRawTransaction", signed, result); err != nil {
		fmt.Println("sendrawtransaction call err", err)
		//panic(err)
		return *result, err
	}

	cnt++
	fmt.Println(fmt.Sprintf("%6d", cnt), "sendrawtransaction hash", result.String())
	return *result, err
}

func getnonce(addr utils.Address) uint64 {
	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		fmt.Println("getnonce diahttp err", err)
		panic(err)
	}
	defer client.Close()

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

func main() {
	signer := types.Signer{}

	flag.StringVar(&rpchost, "u", "http://localhost:8000", "RPC host地址")

	count := flag.Int("n", 3, "输入的投票私钥个数")
	issueHex := flag.String("v", "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032", "投票人私钥")
	voterPass := flag.String("p", "coinbase", "私钥密码")
	value := flag.String("c", "5000000", "要投的URAC数量,单位:(urac)")
	flag.Parse()
	others := flag.Args()

	if len(rpchost) < 7 {
		rpchost = "http://localhost:8000"
	}
	if !strings.Contains(strings.ToLower(rpchost), "http") {
		rpchost = "http://" + rpchost
	}
	if !strings.ContainsRune(string([]rune(rpchost)[len(rpchost)-7:]), ':') {
		rpchost = rpchost + ":8000"
	}

	if len(others) == 0 && *count == 3 {
		others = append(others, "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032")
		others = append(others, "9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658")
		others = append(others, "8605cf6e76c9fc8ac079d0f841bd5e99bd3ad40fdd56af067993ed14fc5bfca8")
	}

	issuerNonce := uint64(0)
	issueValue := new(big.Int)
	issueValue.SetString(*value, 0)
	issueValue = new(big.Int).Mul(big.NewInt(1e18), issueValue)
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(10000000000)

	// issuer
	issuerPriv, _ := crypto.HexToECDSA(*issueHex)
	issuer := crypto.PubkeyToAddress(issuerPriv.PublicKey)
	issuerNonce = getnonce(issuer)

	fmt.Println("issue:", issuer, " Nonce:", issuerNonce)
	fmt.Println("count:", *count, "password:", *voterPass)
	fmt.Println("host:", rpchost)
	//fmt.Println("candidates:", others)

	if len(others) < *count {
		println("输入的投票私钥个数不对")
		return
	}

	for i := 0; i < *count; i++ {
		if len(others[i]) != 64 {
			println("输入的投票私钥长度[", others[i], "]", len(others[i]), " != 64")
			return
		}
	}

	producers := map[utils.Address]*ecdsa.PrivateKey{}
	for i := 0; i < *count; i++ {
		key, err := crypto.HexToECDSA(others[i])
		if err != nil {
			println("私钥[", others[i], "]分析失败：", err.Error())
			return
		}

		addr := crypto.PubkeyToAddress(key.PublicKey)
		producers[addr] = key
		fmt.Println("addr:[", addr.String(), "] PrivKey:[", hex.EncodeToString(crypto.ByteFromECDSA(key)), "]")
	}

	fmt.Println("\r\n\r\n\r\n")
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~我是很浪的波浪线~~~~~~~~~~~~~~~~~~~~~~~~")

	// transfer producers
	issuerNonce = getnonce(issuer)
	for addr := range producers {
		fmt.Println("Tx(Binary) from [", issuer.String(), "] to:[", addr.String(), "] nonce:", issuerNonce)
		if issuer == addr {
			continue
		}
		txTransfer := types.NewTransaction(types.Binary, issuerNonce, issueValue, gasLimit, gasPrice, nil, []*utils.Address{&addr}...)
		txTransfer.SignTx(signer, issuerPriv)
		result, err := sendrawtransaction(txTransfer)
		if err != nil {
			fmt.Println("sendrawtransaction failed:", err)
		} else {
			fmt.Println("Tx Hash:", result.String())
			issuerNonce++
		}
	}
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~我是很浪的波浪线~~~~~~~~~~~~~~~~~~~~~~~~")
	time.Sleep(6 * time.Second)

	// vote producers
	for addr, priv := range producers {
		nonce := getnonce(addr)
		fmt.Println("Tx(LoginCandidate) from [", addr.String(), "] to:[", addr, "] nonce:", nonce)
		txReg := types.NewTransaction(types.LoginCandidate, nonce, big.NewInt(0), gasLimit, gasPrice, nil)
		txReg.SignTx(signer, priv)
		result, err := sendrawtransaction(txReg)
		if err != nil {
			fmt.Println("sendrawtransaction failed:", err)
		} else {
			fmt.Println("Tx Hash:", result.String())
		}
	}

	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~我是很浪的波浪线~~~~~~~~~~~~~~~~~~~~~~~~")
	time.Sleep(6 * time.Second)

	val := new(big.Int).Sub(issueValue, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10))) // 保留10urac作为手续费
	for addr, priv := range producers {
		nonce := getnonce(addr)
		fmt.Println("Tx(Delegate) from [", addr.String(), "] to:[", addr.String(), "] nonce:", nonce)
		txVote := types.NewTransaction(types.Delegate, nonce, val, gasLimit, gasPrice, nil, []*utils.Address{&addr}...)
		txVote.SignTx(signer, priv)
		result, err := sendrawtransaction(txVote)
		if err != nil {
			fmt.Println("sendrawtransaction failed:", err)
		} else {
			fmt.Println("Tx Hash:", result.String())
		}
	}

	fmt.Println("Dpos启动完成.")
}
