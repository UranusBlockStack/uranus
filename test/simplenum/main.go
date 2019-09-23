package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"math/big"
	"math/rand"
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
	cnt     = 0
)

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
		sender, _ := tx.Sender(types.Signer{})
		fmt.Println("sendrawtransaction call err", err, sender, tx.Value(), tx.Type())
		panic(err)
	}

	cnt++
	fmt.Println(fmt.Sprintf("%6d", cnt), "sendrawtransaction hash", result.String(), "type", tx.Type())

	time.Sleep(5 * time.Second)
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

type account struct {
	addr  utils.Address
	priv  *ecdsa.PrivateKey
	nonce uint64
	t     time.Time
}

func main() {
	signer := types.Signer{}

	rpcHost := flag.String("h", "http://localhost:8000", "RPC host地址")
	tcnum := flag.Int("n", 0, "candidate num")

	flag.Parse()
	rpchost = *rpcHost

	issuePrivHex := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	issuerNonce := uint64(0)
	issueValue := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100))
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(10000000000)
	rand.Seed(time.Now().UnixNano())

	// issuer
	issuerPriv, _ := crypto.HexToECDSA(issuePrivHex)
	issuer := crypto.PubkeyToAddress(issuerPriv.PublicKey)
	fmt.Println("issuer", issuer)
	issuerNonce = getnonce(issuer)

	num := 10
	accts := []*account{}
	for i := 0; i < num; i++ {
		priv, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(priv.PublicKey)
		accts = append(accts, &account{
			priv: priv,
			addr: addr,
		})
	}
	candidates := map[utils.Address]*account{}

	for {
		// Transfer
		for _, acct := range accts {
			txTransfer := types.NewTransaction(types.Binary, issuerNonce, new(big.Int).Mul(issueValue, big.NewInt(int64(rand.Intn(5)+1))), gasLimit, gasPrice, nil, []*utils.Address{&acct.addr}...)
			txTransfer.SignTx(signer, issuerPriv)
			sendrawtransaction(txTransfer)
			issuerNonce++
		}

		// Reg
		acct := accts[rand.Intn(len(accts))]
		if _, ok := candidates[acct.addr]; !ok {
			if len(candidates) < num/3 {
				txReg := types.NewTransaction(types.LoginCandidate, acct.nonce, big.NewInt(0), gasLimit, gasPrice, nil)
				txReg.SignTx(signer, acct.priv)
				sendrawtransaction(txReg)
				acct.nonce++

				candidates[acct.addr] = acct
			}
		}

		// UnReg
		if len(candidates) > 5 && rand.Intn(len(accts))%2 == 0 {
			for _, acct := range candidates {
				txUnReg := types.NewTransaction(types.LogoutCandidate, acct.nonce, big.NewInt(0), gasLimit, gasPrice, nil)
				txUnReg.SignTx(signer, acct.priv)
				sendrawtransaction(txUnReg)
				acct.nonce++

				delete(candidates, acct.addr)
				break
			}
		}

		// Vote
		cnum := *tcnum
		if cnum <= 0 {
			cnum = rand.Intn(5)
		}
		caddrs := []*utils.Address{}
		for _, acct := range candidates {
			caddrs = append(caddrs, &acct.addr)
		}
		if len(caddrs) > cnum {
			caddrs = caddrs[:cnum]
		}

		for _, acct := range accts {
			if _, ok := candidates[acct.addr]; !ok {
				if acct.t.IsZero() {
					if len(caddrs) > 0 {
						value := new(big.Int).Div(issueValue, big.NewInt(int64((rand.Intn(2)+1)*2)))
						txVote := types.NewTransaction(types.Delegate, acct.nonce, value, gasLimit, gasPrice, nil, caddrs...)
						txVote.SignTx(signer, acct.priv)
						sendrawtransaction(txVote)
						acct.nonce++

						if rand.Intn(len(accts))%2 == 0 {
							txUnvote := types.NewTransaction(types.UnDelegate, acct.nonce, value, gasLimit, gasPrice, nil)
							txUnvote.SignTx(signer, acct.priv)
							sendrawtransaction(txUnvote)
							acct.nonce++

							acct.t = time.Now()
						}
					}
				} else if time.Now().Sub(acct.t) > 72*time.Hour {
					if rand.Intn(len(accts))%2 == 0 {
						txReem := types.NewTransaction(types.Redeem, acct.nonce, big.NewInt(0), gasLimit, gasPrice, nil)
						txReem.SignTx(signer, acct.priv)
						sendrawtransaction(txReem)
						acct.nonce++

						acct.t = time.Time{}
					}
				}
			}
		}
	}
}
