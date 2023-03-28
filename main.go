package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	zksync2 "github.com/zksync-sdk/zksync2-go"
)

var conf = new(Config)

type Config struct {
	AccountPk string `json:"account_pk"`
	ZkUrl     string `json:"zk_url"`
	EthUrl    string `json:"eth_url"`
	ZkChainId int64  `json:"zk_chain_id"`
}

func initialize() {
	raw, err := os.ReadFile("config.json")
	if err != nil {
		panic(fmt.Sprintf("read config file failed, err: %v", err))
	}

	if err = json.Unmarshal(raw, conf); err != nil {
		panic(fmt.Sprintf("unmarshal config failed, err: %v", err))
	}
}

func main() {
	initialize()

	instance, err := newInstance()
	if err != nil {
		panic(fmt.Sprintf("generate instance failed, err: %v", err))
	}

	//instance.Deposit()
	//instance.Transfer()
	instance.Withdrawal()
}

type Instance struct {
	signer      *zksync2.DefaultEthSigner
	wallet      *zksync2.Wallet
	geth        *rpc.Client
	zkProvider  *zksync2.DefaultProvider
	ethProvider zksync2.EthProvider
}

// deposit signer deposit native ETH1 from l1 to some address at l2
func (ins *Instance) Deposit() {
	split("deposit", "signer deposit l1 ether to l2, and the asset will be locked in contract of diamondProxy")

	to := ins.signer.GetAddress()
	amount := big.NewInt(1000000000000000)
	balance1, _ := ins.wallet.GetBalanceOf(to, zksync2.CreateETH(), zksync2.BlockNumberCommitted)

	tx, err := ins.ethProvider.Deposit(zksync2.CreateETH(), amount, to, nil)
	if err != nil {
		panic(fmt.Sprintf("deposit failed, err: %v", err))
	} else {
		fmt.Printf("deposit success %s\r\n", tx.Hash().Hex())
	}

	time.Sleep(5 * time.Second)
	balance2, _ := ins.wallet.GetBalanceOf(to, zksync2.CreateETH(), zksync2.BlockNumberCommitted)
	fmt.Printf("before deposit amount %s, after deposit amount %s\r\n", balance1.String(), balance2.String())
}

func (ins *Instance) Transfer() {
	split("transfer", "signer transfer mirror asset `ether` on l2")

	to := common.HexToAddress("0x5Ed9a6713962f04DA057e6A949394e002855DF72")
	amount := big.NewInt(1000000000000000)
	balance1, _ := ins.wallet.GetBalanceOf(to, zksync2.CreateETH(), zksync2.BlockNumberCommitted)

	hash, err := ins.wallet.Transfer(to, amount, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("transfer failed, err: %v", err))
	} else {
		fmt.Printf("transfer success %s\r\n", hash.Hex())
	}

	time.Sleep(5 * time.Second)
	receipt, _ := ins.zkProvider.GetTransactionReceipt(hash)
	fmt.Printf("receipt.to %s, dest contract address is %s, L1BatchNumber %s, L1BatchTransactionIndex %s \r\n",
		receipt.To.Hex(), receipt.ContractAddress,
		receipt.L1BatchNumber.String(),
		receipt.L1BatchTxIndex.String(),
	)

	balance2, _ := ins.wallet.GetBalanceOf(to, zksync2.CreateETH(), zksync2.BlockNumberCommitted)
	fmt.Printf("before transfer amount %s, after transfer amount %s\r\n", balance1.String(), balance2.String())
}

func (ins *Instance) Withdrawal() {
	split("withdrawal", "signer withdraw asset from l2 to l1")

	receipt := ins.signer.GetAddress()
	amount := big.NewInt(1000000000000)
	balance1, _ := ins.wallet.GetBalanceOf(receipt, zksync2.CreateETH(), zksync2.BlockNumberCommitted)

	hash, err := ins.wallet.Withdraw(receipt, amount, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("withdrawal failed, err: %v", err))
	} else {
		fmt.Printf("withdrawal succeed %s\r\n", hash.Hex())
	}

	time.Sleep(5 * time.Second)
	balance2, _ := ins.wallet.GetBalanceOf(receipt, zksync2.CreateETH(), zksync2.BlockNumberCommitted)
	fmt.Printf("before withdraw amount %s, after withdraw amount %s\r\n", balance1.String(), balance2.String())
}

func newInstance() (ins *Instance, err error) {
	var pkBytes []byte
	ins = &Instance{}

	if pkBytes, err = hexutil.Decode(conf.AccountPk); err != nil {
		return
	}

	// or from raw PrivateKey bytes
	if ins.signer, err = zksync2.NewEthSignerFromRawPrivateKey(pkBytes, conf.ZkChainId); err != nil {
		return
	}

	// also, init ZkSync Provider, specify ZkSync2 RPC URL (e.g. testnet)
	if ins.zkProvider, err = zksync2.NewDefaultProvider(conf.ZkUrl); err != nil {
		return
	}

	// then init Wallet, passing just created Ethereum Signer and ZkSync Provider
	if ins.wallet, err = zksync2.NewWallet(ins.signer, ins.zkProvider); err != nil {
		return
	}

	// init default RPC client to Ethereum node (Goerli network in case of ZkSync2 testnet)
	if ins.geth, err = rpc.Dial(conf.EthUrl); err != nil {
		return
	}

	// and use it to create Ethereum Provider by Wallet
	if ins.ethProvider, err = ins.wallet.CreateEthereumProvider(ins.geth); err != nil {
		return
	}

	return
}

func split(name, desc string) {
	fmt.Println("=================================================================================================")
	fmt.Println("test", name, ":")
	fmt.Println(desc)
	fmt.Println("")
	fmt.Println("")
}
