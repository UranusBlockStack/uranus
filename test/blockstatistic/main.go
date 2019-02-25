package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/UranusBlockStack/uranus/common/utils"
	urpc "github.com/UranusBlockStack/uranus/rpc"
	"github.com/UranusBlockStack/uranus/rpcapi"
)

var (
	rpchost        = "http://localhost:8000"
	cnt     uint64 = 0

	startH = int64(0)
	endH   = int64(-1)
)

func testGetLatestBlockHeight() (latest int64) {
	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		//fmt.Println("testGetBlockInfo-DialHTTP err", err)
		return 0
	}
	defer client.Close()

	result := new(utils.Uint64)
	err = client.Call("BlockChain.GetLatestBlockHeight", nil, &result)
	if err != nil {
		//fmt.Println("testGetBlockInfo-Call err", err)
		return 0
	}

	return (int64)(*result)
}

func testGetBFTBlockHeight() (bft int64) {
	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		//fmt.Println("testGetBlockInfo-DialHTTP err", err)
		return 0
	}
	defer client.Close()

	result := new(utils.Big)
	err = client.Call("Dpos.GetConfirmedBlockNumber", nil, &result)
	if err != nil {
		//fmt.Println("testGetBlockInfo-Call err", err)
		return 0
	}
	str := result.String()

	bft, err = strconv.ParseInt(str, 0, 64)
	if err != nil {
		bft = 0
		panic(err)
	}
	return
}


func testGetBlockInfo(h int64, time *uint64) (addr string, cntTxs int64, sizeBlock uint, err error) {
	req := rpcapi.GetBlockByHeightArgs{}
	req.BlockHeight = new(rpcapi.BlockHeight)
	*req.BlockHeight = rpcapi.BlockHeight(h)
	req.FullTx = false

	client, err := urpc.DialHTTP(rpchost)
	if err != nil {
		//fmt.Println("testGetBlockInfo-DialHTTP err", err)
		return "", 0, 0, err
	}

	result := map[string]interface{}{}
	err = client.Call("BlockChain.GetBlockByHeight", req, &result)
	if err != nil {
		//fmt.Println("testGetBlockInfo-Call err", err)
		return "", 0, 0, err
	}
	defer client.Close()

	var b bool
	for name, item := range result {
		if name == "miner" {
			//fmt.Println(name, item, reflect.TypeOf(item))
			addr, b = item.(string)
			if !b {
				addr = "ERROR"
			}
			continue
		}
		if name == "size" {
			//fmt.Println(name, item, reflect.TypeOf(item))
			strSizeBlock, b := item.(string)
			if b {
				si, err := strconv.ParseInt(strSizeBlock, 0, 32)
				if err == nil {
					sizeBlock = uint(si)
				}
			}
			continue
		}
		if time != nil && name == "timestamp" {
			//fmt.Println(name, item, reflect.TypeOf(item))
			strTime, b := item.(string)
			if b {
				*time, err = strconv.ParseUint(strTime, 0, 64)
				if err != nil {
					*time = 0
				}
			}
			continue
		}
		if name == "transactions" {
			//fmt.Println(name, item, reflect.TypeOf(item))
			i, b := item.([]interface{})
			if b {
				cntTxs = int64(len(i))
			}
			continue
		}
	}

	return
}

//格式化数值    1,234,567,898.55
func NumberFormat(str string) string {
	length := len(str)
	if length < 4 {
		return str
	}
	arr := strings.Split(str, ".") //用小数点符号分割字符串,为数组接收
	length1 := len(arr[0])
	if length1 < 4 {
		return str
	}
	count := (length1 - 1) / 3
	for i := 0; i < count; i++ {
		arr[0] = arr[0][:length1-(i+1)*3] + "," + arr[0][length1-(i+1)*3:]
	}
	return strings.Join(arr, ".") //将一系列字符串连接为一个字符串，之间用sep来分隔。
}
func NumberFormatInt(str int64) string {
	i := strconv.FormatInt(str, 10)
	return NumberFormat(i)
}

func main() {
	rpcHost := flag.String("u", "http://localhost:8000", "RPC host地址")
	_start := flag.Int64("s", 1, "统计的起始高度")
	_end := flag.Int64("e", -1, "统计的结束高度")
	bDetails := flag.Bool("d", false, "是否显示区块细节情况")
	validators := flag.Int64("v", 3, "出块人个数")
	intervalTime := flag.Int64("i", 500, "出块时间间隔(单位毫秒)")
	blockRepeat := flag.Int64("r", 12, "单个节点一次出块个数")
	flag.Parse()

	if len(*rpcHost) < 7 {
		*rpcHost = "http://localhost:8000"
	}
	if !strings.Contains(strings.ToLower(*rpcHost), "http") {
		*rpcHost = "http://" + *rpcHost
	}
	if !strings.ContainsRune(string([]rune(*rpcHost)[len(*rpcHost)-7:]), ':') {
		*rpcHost = *rpcHost + ":8000"
	}

	if *intervalTime <= 0 {
		*intervalTime = 500
	}
	if *blockRepeat <= 0 {
		*blockRepeat = 12
	}
	if *validators <= 0 {
		*validators = 3
	}

	startH = *_start
	endH = *_end
	bftH := testGetBFTBlockHeight()
	rpchost = *rpcHost
	intervaltime := uint64(*intervalTime * int64(time.Millisecond))
	epoch := uint64(*validators*(*blockRepeat)) * intervaltime

	if endH == -1 {
		endH = testGetLatestBlockHeight()
	}
	if endH == 0 {
		println("Can`t get block end height.")
		return
	}

	var timeStart, timeEnd, timeStamp, privts uint64
	var totalTxsCount, totalBlockSize int64
	var maxTxs int64
	var maxSize int64

	for i := startH; i <= endH; i++ {
		var addr string
		var cntTxs int64
		var sizeBlock uint
		var err error

		addr, cntTxs, sizeBlock, err = testGetBlockInfo(i, &timeStamp)

		if timeStart == 0 {
			timeStart = timeStamp
			privts = timeStamp
		} else if i == endH {
			timeEnd = timeStamp
		}

		if cntTxs < 0 || sizeBlock <= 0 || err != nil {
			panic(err)
		}

		if cntTxs > maxTxs {
			maxTxs = cntTxs
		}

		if int64(sizeBlock) > maxSize {
			maxSize = int64(sizeBlock)
		}

		totalTxsCount += (cntTxs)
		totalBlockSize += int64(sizeBlock)

		printDetails := func(newaddr string, ch int64) {
			if (timeStamp%epoch)/(uint64(*blockRepeat)*intervaltime) != ((timeStamp-intervaltime)%epoch)/(uint64(*blockRepeat)*intervaltime) {
				fmt.Printf("\n%s:", newaddr)
			}

			if ch == -1 {
				fmt.Printf(" _")
			} else if ch == 0 {
				fmt.Printf(" 0")
			} else if ch == 1 {
				fmt.Printf(" 1")
			} else if ch == 2 {
				fmt.Printf(" 2")
			} else if ch == 3 {
				fmt.Printf(" 3")
			} else if ch == 4 {
				fmt.Printf(" 4")
			} else if ch == 5 {
				fmt.Printf(" 5")
			} else if ch == 6 {
				fmt.Printf(" 6")
			} else if ch == 7 {
				fmt.Printf(" 8")
			} else if ch == 9 {
				fmt.Printf(" 9")
			} else if ch == 10 {
				fmt.Printf(" A")
			} else if ch == 11 {
				fmt.Printf(" B")
			} else if ch == 12 {
				fmt.Printf(" C")
			} else if ch == 13 {
				fmt.Printf(" D")
			} else if ch == 14 {
				fmt.Printf(" E")
			} else if ch == 15 {
				fmt.Printf(" F")
			} else if ch <= 20 {
				fmt.Printf(" S")
			} else if ch <= 50 {
				fmt.Printf(" M")
			} else if ch <= 100 {
				fmt.Printf(" L")
			} else if ch <= 200 {
				fmt.Printf("&1")
			} else if ch <= 300 {
				fmt.Printf("&2")
			} else if ch <= 400 {
				fmt.Printf("&3")
			} else if ch <= 500 {
				fmt.Printf("&4")
			} else if ch <= 600 {
				fmt.Printf("&5")
			} else if ch <= 700 {
				fmt.Printf("&6")
			} else if ch <= 800 {
				fmt.Printf("&7")
			} else if ch <= 900 {
				fmt.Printf("&8")
			} else if ch <= 1000 {
				fmt.Printf("&9")
			} else if ch <= 1100 {
				fmt.Printf("$0")
			} else if ch <= 1200 {
				fmt.Printf("$1")
			} else if ch <= 1300 {
				fmt.Printf("$2")
			} else if ch <= 1400 {
				fmt.Printf("$3")
			} else if ch <= 1500 {
				fmt.Printf("$4")
			} else if ch <= 1600 {
				fmt.Printf("$5")
			} else if ch <= 1700 {
				fmt.Printf("$6")
			} else if ch <= 1800 {
				fmt.Printf("$7")
			} else if ch <= 1900 {
				fmt.Printf("$8")
			} else if ch <= 2000 {
				fmt.Printf("$9")
			} else {
				fmt.Printf(" *")
			}
		}

		if *bDetails {

			ttimestamp := timeStamp
			for ttimestamp-privts >= 2*intervaltime {

				if ttimestamp-privts >= epoch {
					count := uint64((ttimestamp-privts)/epoch)
					timeStamp = privts + (count * epoch)
					fmt.Println("\r\n Lost several epoch(s):" , count)
				}else{
					timeStamp = privts + intervaltime
					if (timeStamp%epoch)%(uint64(*blockRepeat)*intervaltime) == 0 && ttimestamp-timeStamp >= uint64(*blockRepeat)*intervaltime {
						printDetails(utils.Address{}.String(), -1)
					} else {
						printDetails(addr, -1)
					}
				}

				privts = timeStamp
			}
			timeStamp = ttimestamp

			privts = timeStamp
			printDetails(addr, cntTxs)

			if i == bftH {
				fmt.Printf("{BFT}")
			}
		}
	}

	fmt.Printf("\n")

	var ts int64 = int64(timeStart)
	tStart := time.Unix(ts/int64(time.Second), ts%int64(time.Second))
	ts = int64(timeEnd)
	tEnd := time.Unix(ts/int64(time.Second), ts%int64(time.Second))

	if timeEnd < timeStart {
		panic("启动时间大于结束时间")
	}
	timeElapse := int64(timeEnd-timeStart) / int64(time.Second)
	if timeElapse == 0 {
		timeElapse = 1
	}

	fmt.Println("Block Num:", startH, "-", endH)
	fmt.Println("Time From:", tStart)
	fmt.Println("Time To  :", tEnd)
	fmt.Println("Elapse   :", timeElapse/(60*60), "H:", timeElapse%(60*60)/60, "M:", timeElapse%(60*60)%60, "S")
	fmt.Println("Block Max Txs:", maxTxs)
	fmt.Println("Block Max Size:", maxSize)
	fmt.Println("Total Txs:", NumberFormatInt(totalTxsCount), "\t\tSpeed:", float64(totalTxsCount)/float64(timeElapse), "Tx/s")
	fmt.Println("Total Len:", NumberFormatInt(totalBlockSize), "B", "\tSpeed:", float64(totalBlockSize)/float64(timeElapse), "B/s")
}
