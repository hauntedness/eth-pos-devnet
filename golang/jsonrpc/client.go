package jsonrpc

import (
	"context"
	"encoding/json"
	"math/big"
	"math/rand"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/hauntedness/httputil"
	"github.com/pkg/errors"
)

const (
	Wei   = params.Wei
	GWei  = params.GWei
	Ether = params.Ether
)

var (
	eth_client *ethclient.Client
	rpc_client *rpc.Client
)

func init() {
	var err error
	rpc_client, err = rpc.DialHTTP("http://localhost:8545")
	if err != nil {
		panic(err)
	}
	eth_client = ethclient.NewClient(rpc_client)
}

func EthAccounts() ([]string, error) {
	payload := `{"jsonrpc":"2.0", "method":"eth_accounts","params":[], "id":1}`
	res, err := httputil.Post("http://localhost:8545", strings.NewReader(payload), nil)
	if err != nil {
		return nil, err
	}
	type EthAccountsResponse struct {
		Jsonrpc string   `json:"jsonrpc"`
		ID      int64    `json:"id"`
		Result  []string `json:"result"`
	}
	var ethAccountsResponse EthAccountsResponse
	err = json.Unmarshal(res, &ethAccountsResponse)
	if err != nil {
		return nil, err
	}
	return ethAccountsResponse.Result, nil
}

func EthGetBalanceFloat(address string, block string) (*big.Float, error) {
	var result *big.Float
	var err error
	if block == "" {
		block = "latest"
	}
	eth_client.BalanceAt(context.TODO(), common.HexToAddress(address), nil)
	err = rpc_client.Call(&result, "eth_getBalance", address, block)
	return result, err
}

func GetPrivateKeyFromJsonBytes(json_in_bytes []byte, auth string) (*keystore.Key, error) {
	key, err := keystore.DecryptKey(json_in_bytes, auth)
	if err != nil {
		return nil, err
	}
	key.PrivateKey.Public()
	return key, nil
}

func GetPrivateKeyFromPath(path string, auth string) (*keystore.Key, error) {
	json_in_bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return GetPrivateKeyFromJsonBytes(json_in_bytes, auth)
}

type opt struct {
	gasPrice *big.Int
	gasLimit uint64
	nonce    uint64
	data     []byte
}
type SendTxnOption func(*opt)

// Deprecated: use EthSendTx instead
func EthSendTransaction(ks *keystore.KeyStore, auth string, from accounts.Account, to accounts.Account, amount *big.Int, options ...SendTxnOption) error {
	err := ks.Unlock(from, auth)
	if err != nil {
		return errors.Wrap(err, "can not unlock account:"+from.Address.Hex())
	}
	balance, err := eth_client.BalanceAt(context.TODO(), from.Address, nil)
	if err != nil {
		return errors.WithStack(err)
	} else if balance.Cmp(amount) <= 0 {
		return errors.New("no enough balance, balance: " + balance.String())
	}
	var opt opt
	for _, sto := range options {
		sto(&opt)
	}
	if opt.nonce == 0 {
		opt.nonce, err = eth_client.PendingNonceAt(context.TODO(), from.Address)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	if opt.gasPrice == nil || opt.gasPrice.Cmp(big.NewInt(0)) <= 0 {
		opt.gasPrice, err = eth_client.SuggestGasPrice(context.TODO())
		if err != nil {
			return errors.WithStack(err)
		}
	}
	if opt.gasLimit == 0 {
		opt.gasLimit = 21000
	}
	if len(opt.data) == 0 {
		opt.data = make([]byte, 0)
		_, _ = rand.Read(opt.data)
	}
	txn := types.NewTransaction(opt.nonce, to.Address, amount, opt.gasLimit, opt.gasPrice, opt.data)
	chainId, err := eth_client.ChainID(context.TODO())
	if err != nil {
		return errors.WithStack(err)
	}
	signed_txn, err := ks.SignTx(from, txn, chainId)
	if err != nil {
		return errors.WithStack(err)
	}
	err = eth_client.SendTransaction(context.TODO(), signed_txn)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

type SetTxnData func(*types.DynamicFeeTx)

func EthSendDynamicFeeTx(ks *keystore.KeyStore, auth string, from accounts.Account, to accounts.Account, amount *big.Int, fn ...SetTxnData) error {
	err := ks.Unlock(from, auth)
	if err != nil {
		return errors.Wrap(err, "can not unlock account:"+from.Address.Hex())
	}
	balance, err := eth_client.BalanceAt(context.TODO(), from.Address, nil)
	if err != nil {
		return errors.WithStack(err)
	} else if balance.Cmp(amount) <= 0 {
		return errors.New("no enough balance, balance: " + balance.String())
	}
	chainId, err := eth_client.ChainID(context.TODO())
	if err != nil {
		return errors.WithStack(err)
	}
	var data = types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     0,
		GasTipCap: nil,
		GasFeeCap: nil,
		Gas:       0,
		To:        (*common.Address)(to.Address.Bytes()),
		Value:     amount,
		Data:      []byte{},
	}
	for i := range fn {
		fn[i](&data)
	}

	if data.Nonce == 0 {
		data.Nonce, err = eth_client.PendingNonceAt(context.TODO(), from.Address)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	if data.Gas == 0 {
		v, err := eth_client.SuggestGasPrice(context.TODO())
		if err != nil {
			return errors.WithStack(err)
		}
		data.Gas = uint64(v.Int64())
	}
	if data.GasFeeCap == nil {
		data.GasFeeCap, err = eth_client.SuggestGasTipCap(context.TODO())
		if err != nil {
			return errors.WithStack(err)
		}
	}
	txn := types.NewTx(&data)
	signed_txn, err := ks.SignTx(from, txn, chainId)
	if err != nil {
		return errors.WithStack(err)
	}
	err = eth_client.SendTransaction(context.TODO(), signed_txn)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
