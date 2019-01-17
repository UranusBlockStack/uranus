package main

import (
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/UranusBlockStack/uranus/common/crypto"
)

func main() {
	count := flag.Int("count", 1, "how many private key(s) produced.")
	flag.Parse()
	for i := 0; i < *count; i++ {
		priv, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(priv.PublicKey)

		fmt.Println("produce[", i, "]:", addr, hex.EncodeToString(crypto.ByteFromECDSA(priv)))
	}
}
