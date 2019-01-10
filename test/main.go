package main

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	urpc "github.com/UranusBlockStack/uranus/rpc"
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

func main() {
	signer := types.Signer{}
	issuePrivHex := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	issuerNonce := uint64(0)
	issueValue := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100000000))
	producersSize := 21
	votersSize := 1
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(10000000000)
	wokers := 1

	// issuer
	issuerPriv, _ := crypto.HexToECDSA(issuePrivHex)
	issuer := crypto.PubkeyToAddress(issuerPriv.PublicKey)
	fmt.Println("issuer", issuer)

	transfers := []*utils.Address{}
	// generate producers
	producers := map[utils.Address]*ecdsa.PrivateKey{}
	for i := 0; i < producersSize; i++ {
		priv, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(priv.PublicKey)
		producers[addr] = priv
		transfers = append(transfers, &addr)
	}

	// generate voters
	tps := []*ecdsa.PrivateKey{}
	voters := map[utils.Address]*ecdsa.PrivateKey{}
	for i := 0; i < votersSize; i++ {
		priv, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(priv.PublicKey)
		voters[addr] = priv
		transfers = append(transfers, &addr)
		tps = append(tps, priv)
	}

	// transfers
	for _, addr := range transfers {
		// transfer
		txTransfer := types.NewTransaction(types.Binary, issuerNonce, issueValue, gasLimit, gasPrice, nil, []*utils.Address{addr}...)
		txTransfer.SignTx(signer, issuerPriv)
		sendrawtransaction(txTransfer)
		issuerNonce++
	}
	issuerNonce--

	time.Sleep(6 * time.Second)
	// reg & unreg producer
	i := 0
	validateProducers := []*utils.Address{}
	for addr, priv := range producers {
		val := new(big.Int).Div(issueValue, big.NewInt(2))
		// reg producers
		txReg := types.NewTransaction(types.LoginCandidate, 0, big.NewInt(0), gasLimit, gasPrice, nil, []*utils.Address{}...)
		txReg.SignTx(signer, priv)
		sendrawtransaction(txReg)

		if i%2 > 0 {
			// unreg
			txUnReg := types.NewTransaction(types.LogoutCandidate, 1, big.NewInt(0), gasLimit, gasPrice, nil, []*utils.Address{}...)
			sendrawtransaction(txUnReg)
		} else {
			txVote := types.NewTransaction(types.Delegate, 1, val, gasLimit, gasPrice, nil, validateProducers...)
			txVote.SignTx(signer, priv)
			sendrawtransaction(txVote)
			validateProducers = append(validateProducers, &addr)
		}
	}
	wg := &sync.WaitGroup{}
	wg.Add(wokers)
	cnt := len(tps) / wokers
	for i := 0; i < wokers; i++ {
		f := i * cnt
		t := (i + 1) * cnt
		if t > len(tps) {
			t = len(tps)
		}
		ttps := tps[f:t]
		go func(tps []*ecdsa.PrivateKey) {
			defer wg.Done()
			nonce := uint64(0)
			for {
				for _, priv := range tps {
					tpriv, _ := crypto.GenerateKey()
					addr := crypto.PubkeyToAddress(tpriv.PublicKey)

					val := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10))
					// transfer
					txTransfer := types.NewTransaction(types.Binary, nonce, val, gasLimit, gasPrice, nil, []*utils.Address{&addr}...)
					txTransfer.SignTx(signer, priv)
					sendrawtransaction(txTransfer)
					nonce++

					n := len(validateProducers)
					// vote
					txVote := types.NewTransaction(types.Delegate, nonce, new(big.Int).Div(val, big.NewInt(2)), gasLimit, gasPrice, nil, validateProducers[:n*2/3]...)
					txVote.SignTx(signer, priv)
					sendrawtransaction(txVote)
					nonce++

					// // unvote
					// txUnvote := types.NewTransaction(types.UnDelegate, nonce, big.NewInt(0), gasLimit, gasPrice, nil, validateProducers[:n*1/3]...)
					// txUnvote.SignTx(signer, priv)
					// sendrawtransaction(txUnvote)
					// nonce++
				}
			}
		}(ttps)
	}
	wg.Wait()
}
