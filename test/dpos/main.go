package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
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

func sendrawtransaction(tx *types.Transaction) {
	// tjson, _ := json.Marshal(tx)
	// fmt.Println("sendrawtransaction content", string(tjson))

	bts, _ := rlp.EncodeToBytes(tx)
	signed := "0x" + utils.BytesToHex(bts)

	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		fmt.Println("sendrawtransaction diahttp err", err)
		panic(err)
	}

	result := &utils.Hash{}
	if err := client.Call("Uranus.SendRawTransaction", signed, result); err != nil {
		fmt.Println("sendrawtransaction call err", err)
		panic(err)
	}

	cnt++
	fmt.Println(fmt.Sprintf("%6d", cnt), "sendrawtransaction hash", result.String())
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

func main() {
	// 1. generate producers
	// 2. transfer producers
	// 3. reg & vote producers
	// 4. dpos starting
	signer := types.Signer{}
	issuePrivHex := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	issuerNonce := uint64(0)
	issueValue := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100000000))
	producersSize := 5
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(10000000000)
	// issuer
	issuerPriv, _ := crypto.HexToECDSA(issuePrivHex)
	issuer := crypto.PubkeyToAddress(issuerPriv.PublicKey)
	fmt.Println("issuer", issuer)
	issuerNonce = getnonce(issuer)

	// generate producers
	producers := map[utils.Address]*ecdsa.PrivateKey{}
	for i := 0; i < producersSize; i++ {
		priv, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(priv.PublicKey)
		producers[addr] = priv
		fmt.Println("producer", crypto.PubkeyToAddress(priv.PublicKey), hex.EncodeToString(crypto.ByteFromECDSA(priv)))
	}

	// transfer producers
	for addr := range producers {
		txTransfer := types.NewTransaction(types.Binary, issuerNonce, issueValue, gasLimit, gasPrice, nil, []*utils.Address{&addr}...)
		txTransfer.SignTx(signer, issuerPriv)
		sendrawtransaction(txTransfer)
		issuerNonce++
	}
	issuerNonce--

	time.Sleep(6 * time.Second)

	// vote producers
	for addr, priv := range producers {
		nonce := uint64(0)
		txReg := types.NewTransaction(types.LoginCandidate, nonce, big.NewInt(0), gasLimit, gasPrice, nil)
		txReg.SignTx(signer, priv)
		sendrawtransaction(txReg)
		nonce++

		val := new(big.Int).Div(issueValue, big.NewInt(2))
		txVote := types.NewTransaction(types.Delegate, nonce, val, gasLimit, gasPrice, nil, []*utils.Address{&addr, &issuer}...)
		txVote.SignTx(signer, priv)
		sendrawtransaction(txVote)
		nonce++
	}
	// dpos staring
}
