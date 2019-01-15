package main

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/utils"
	urpc "github.com/UranusBlockStack/uranus/rpc"
	"github.com/UranusBlockStack/uranus/rpcapi"
)

var (
	rpchost = "http://localhost:8000"
)

var cnt uint64 = 0

func signandsendrawtransaction(data []byte) {
	cnt++
	fmt.Println(fmt.Sprintf("%6d", cnt), "signandsendrawtransaction content", string(data))
	tx := &rpcapi.SendTxArgs{}
	if err := json.Unmarshal(data, tx); err != nil {
		fmt.Println("signandsendrawtransaction unmarshal err", err)
		panic(err)
	}

	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		fmt.Println("signandsendrawtransaction diahttp err", err)
		panic(err)
	}
	result := &utils.Hash{}
	if err := client.Call("Uranus.SignAndSendTransaction", tx, result); err != nil {
		fmt.Println("signandsendrawtransaction call err", err)
		panic(err)
	}
	fmt.Println(fmt.Sprintf("%6d", cnt), "signandsendrawtransaction hash", result.String())
}

func main() {
	issueValue := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100))
	gasLimit := uint64(90000)
	gasPrice := big.NewInt(10000000000)

	// issuer
	issuer := utils.HexToAddress("0x970E8128AB834E8EAC17Ab8E3812F010678CF791")
	fmt.Println("issuer", issuer)

	priv, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(priv.PublicKey)

	//transfer
	txTransferArgs := fmt.Sprintf(`{"TxType":"0x0","From": "%v","Tos": ["%v"],"Gas": "0x%v", "GasPrice": "0x%v","Value": "0x%v", "Data": "0x00","Passphrase":"coinbase"}`, issuer, addr, big.NewInt(int64(gasLimit)).Text(16), gasPrice.Text(16), issueValue.Text(16))
	signandsendrawtransaction([]byte(txTransferArgs))

	// reg producers tos==nil
	txRegArgs := fmt.Sprintf(`{"TxType":"0x1","From": "%v","Gas": "0x%v", "GasPrice": "0x%v","Value": "0x%v", "Data": "0x00","Passphrase":"coinbase"}`, issuer, big.NewInt(int64(gasLimit)).Text(16), gasPrice.Text(16), big.NewInt(0).Text(16))
	signandsendrawtransaction([]byte(txRegArgs))

	// vote len(tos)<=30
	txVoteArgs := fmt.Sprintf(`{"TxType":"0x3","From": "%v","Tos": ["%v"],"Gas": "0x%v", "GasPrice": "0x%v","Value": "0x%v", "Data": "0x00","Passphrase":"coinbase"}`, issuer, issuer, big.NewInt(int64(gasLimit)).Text(16), gasPrice.Text(16), issueValue.Text(16))
	signandsendrawtransaction([]byte(txVoteArgs))

	// unvote tos == nil
	txUnvoteArgs := fmt.Sprintf(`{"TxType":"0x4","From": "%v","Gas": "0x%v", "GasPrice": "0x%v","Value": "0x%v", "Data": "0x00","Passphrase":"coinbase"}`, issuer, big.NewInt(int64(gasLimit)).Text(16), gasPrice.Text(16), big.NewInt(0).Text(16))
	signandsendrawtransaction([]byte(txUnvoteArgs))

	// unreg tos == nil
	txUnRegArgs := fmt.Sprintf(`{"TxType":"0x2","From": "%v","Gas": "0x%v", "GasPrice": "0x%v","Value": "0x%v", "Data": "0x00","Passphrase":"coinbase"}`, issuer, big.NewInt(int64(gasLimit)).Text(16), gasPrice.Text(16), big.NewInt(0).Text(16))
	signandsendrawtransaction([]byte(txUnRegArgs))

}
