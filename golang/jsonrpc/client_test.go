package jsonrpc

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

var __base_accounts []string

func TestMain(m *testing.M) {
	ks := keystore.NewKeyStore(__keystore_path, keystore.StandardScryptN, keystore.StandardScryptP)
	for _, a := range ks.Accounts() {
		__base_accounts = append(__base_accounts, a.Address.Hex())
	}
	i := m.Run()
	os.Exit(i)
}

func TestEthAccounts(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "test:success",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			accounts, err := EthAccounts()
			if err != nil {
				t.Error(err)
			} else if accounts == nil {
				t.Error("did not get accounts from: jsonrpc.EthAccounts()")
			} else {
				t.Logf("accounts:%s", accounts)
			}
		})
	}
}

func TestEthGetBalance(t *testing.T) {
	type args struct {
		address string
		block   string
	}
	type TestCase struct {
		name    string
		args    args
		want    *big.Float
		wantErr bool
	}
	f, _, err := big.ParseFloat("0", 16, 0, big.ToNearestAway)
	if err != nil {
		t.Error(err)
	}
	tests := []TestCase{
		{
			name: "test get balance",
			args: args{
				address: __base_accounts[0],
				block:   "",
			},
			want:    f,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EthGetBalanceFloat(tt.args.address, tt.args.block)
			if (err != nil) != tt.wantErr {
				t.Errorf("EthGetBalance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Cmp(tt.want) != 0 {
				t.Errorf("EthGetBalance() = %f, want %f", got, tt.want)
			} else {
				t.Logf("EthGetBalance() = %f", got)
			}
		})
	}
}

func TestBalanceAt(t *testing.T) {
	balance, err := eth_client.BalanceAt(context.TODO(), common.HexToAddress(__base_accounts[0]), nil)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%f", balance)
}

func TestChainId(t *testing.T) {
	id, err := eth_client.ChainID(context.TODO())
	if err != nil {
		t.Error(err)
	}
	if id.Cmp(big.NewInt(1337)) != 0 {
		t.Errorf("want 1137, get %s", id)
	}
}

var __keystore_path = filepath.Join("..", "..", "execution", "keystore")

const __password = "test123456"

func TestImportKeyStore(t *testing.T) {
	ks := keystore.NewKeyStore(__keystore_path, keystore.StandardScryptN, keystore.StandardScryptP)
	accounts := ks.Accounts()
	for _, acc := range accounts {
		t.Logf("restore accounts with address hex: %v", acc.Address.Hex())
	}
}

func TestKeyStoreNewAccount(t *testing.T) {
	ks := keystore.NewKeyStore(__keystore_path, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(__password)
	if err != nil {
		t.Error(err)
	}
	t.Logf("generated accounts with address hex: %v", acc.Address.Hex())
}

func TestKeyStorePrivateKey(t *testing.T) {
	ks := keystore.NewKeyStore(__keystore_path, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(__password)
	if err != nil {
		t.Error(err)
	}
	t.Logf("generated accounts with address hex: %v", acc.Address.Hex())
}

func TestBlockHeader(t *testing.T) {
	header, err := eth_client.HeaderByNumber(context.TODO(), nil)
	if err != nil {
		t.Error(err)
	}
	t.Logf("header block: %v", header.Number.String())
}

func TestBlockByNumber(t *testing.T) {
	header, err := eth_client.HeaderByNumber(context.TODO(), big.NewInt(0))
	if err != nil {
		t.Error(err)
	}
	t.Logf("header block: %v", header.Number.String())
}

func TestSendTransaction(t *testing.T) {
	ks := keystore.NewKeyStore(__keystore_path, keystore.StandardScryptN, keystore.StandardScryptP)
	var from_account accounts.Account = ks.Accounts()[1]
	to_account := ks.Accounts()[2]
	var amount *big.Int = big.NewInt(21000)
	err := EthSendTransaction(ks, __password, from_account, to_account, amount)
	if err != nil {
		t.Error(err)
	}
}

func TestEthSendDynamicFeeTx(t *testing.T) {
	ks := keystore.NewKeyStore(__keystore_path, keystore.StandardScryptN, keystore.StandardScryptP)
	var from_account accounts.Account = ks.Accounts()[0]
	to_account := ks.Accounts()[2]
	var amount *big.Int = big.NewInt(21000)
	err := EthSendDynamicFeeTx(ks, __password, from_account, to_account, amount)
	if err != nil {
		t.Error(err)
	}
}
