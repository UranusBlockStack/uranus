package main

import (
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

	time.Sleep(10 * time.Second)
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
	signer := types.Signer{}
	issuePrivHex := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	issuerNonce := uint64(0)
	issueValue := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100000000))
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(10000000000)
	epchos := 1

	// issuer
	issuerPriv, _ := crypto.HexToECDSA(issuePrivHex)
	issuer := crypto.PubkeyToAddress(issuerPriv.PublicKey)
	fmt.Println("issuer", issuer)
	issuerNonce = getnonce(issuer)

	for i := 0; i < epchos; i++ {
		priv, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(priv.PublicKey)

		//transfer
		txTransfer := types.NewTransaction(types.Binary, issuerNonce, issueValue, gasLimit, gasPrice, nil, []*utils.Address{&addr}...)
		txTransfer.SignTx(signer, issuerPriv)
		sendrawtransaction(txTransfer)
		issuerNonce++

		nonce := uint64(0)
		// reg producers
		txReg := types.NewTransaction(types.LoginCandidate, nonce, big.NewInt(0), gasLimit, gasPrice, nil)
		txReg.SignTx(signer, priv)
		sendrawtransaction(txReg)
		nonce++

		// // vote
		// txVote := types.NewTransaction(types.Delegate, nonce, new(big.Int).Div(issueValue, big.NewInt(2)), gasLimit, gasPrice, nil, []*utils.Address{&addr}...)
		// txVote.SignTx(signer, priv)
		// sendrawtransaction(txVote)

		// // unvote
		// txUnvote := types.NewTransaction(types.UnDelegate, nonce, big.NewInt(0), gasLimit, gasPrice, nil)
		// txUnvote.SignTx(signer, priv)
		// sendrawtransaction(txUnvote)
		// nonce++

		// // unreg
		// txUnReg := types.NewTransaction(types.LogoutCandidate, nonce, big.NewInt(0), gasLimit, gasPrice, nil)
		// txUnReg.SignTx(signer, priv)
		// sendrawtransaction(txUnReg)
		// nonce++
	}
}
