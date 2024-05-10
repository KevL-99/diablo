package sui

import (
	"bytes"
	"context"
	"diablo/core"
	"fmt"
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/sui"
	"golang.org/x/crypto/ed25519"
)

type BlockchainBuilder struct {
	logger          core.Logger
	client          sui.ISuiAPI
	ctx             context.Context
	premadeAccounts []account
	usedAccounts    int
}

type account struct {
	address  string
	private  ed25519.PrivateKey
	coinData string
	nonce    uint64
}

func newBuilder(logger core.Logger, client sui.ISuiAPI) *BlockchainBuilder {
	return &BlockchainBuilder{
		logger:          logger,
		client:          client,
		ctx:             context.Background(),
		premadeAccounts: make([]account, 0),
		usedAccounts:    0,
	}
}

func (this *BlockchainBuilder) addAccount(address string, private ed25519.PrivateKey) {
	this.premadeAccounts = append(this.premadeAccounts, account{
		address: address,
		private: private,
		nonce:   0,
	})
}

func (this *BlockchainBuilder) CreateAccount(stake int) (interface{}, error) {
	var ret *account
	var ctx = context.Background()

	if this.usedAccounts < len(this.premadeAccounts) {
		// fund the accounts
		acc := this.premadeAccounts[this.usedAccounts]
		header := map[string]string{}
		err := RequestSuiFromFaucet("http://127.0.0.1:5003/gas", acc.address, header)
		if err != nil {
			fmt.Println(err.Error())
			return nil, fmt.Errorf("error funding account")
		}

		rsp, err := this.client.SuiXGetAllCoins(ctx, models.SuiXGetAllCoinsRequest{
			Owner: acc.address,
			Limit: 5,
		})
		if err != nil {
			fmt.Println(err.Error())
			return nil, fmt.Errorf("error getting coin data")
		}
		coinData := rsp.Data[0].CoinObjectId
		this.premadeAccounts[this.usedAccounts].coinData = coinData
		ret = &this.premadeAccounts[this.usedAccounts]
		this.usedAccounts += 1
	} else {
		return nil, fmt.Errorf("can only use %d premade accounts",
			this.usedAccounts)
	}

	return ret, nil
}

func (this *BlockchainBuilder) CreateContract(name string) (interface{}, error) {
	return nil, fmt.Errorf("CreateContract: not implemented")
}

func (this *BlockchainBuilder) CreateResource(domain string) (core.SampleFactory, bool) {
	return nil, false
}

func (this *BlockchainBuilder) EncodeTransfer(amount int, from, to interface{}, info core.InteractionInfo) ([]byte, error) {
	var tx *transferTransaction
	var buffer bytes.Buffer
	var faccount, taccount *account
	var err error

	faccount = from.(*account)
	taccount = to.(*account)

	tx = newTransferTransaction(faccount.nonce, uint64(amount), faccount.private, faccount.address, taccount.address, faccount.coinData)

	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	faccount.nonce += 1
	// fmt.Printf(string(buffer.Bytes()))
	return buffer.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInvoke(from interface{}, contract interface{}, function string, info core.InteractionInfo) ([]byte, error) {
	return nil, fmt.Errorf("EncodeInvoke: not implemented")
}

func (this *BlockchainBuilder) EncodeInteraction(itype string, expr core.BenchmarkExpression, info core.InteractionInfo) ([]byte, error) {
	return nil, fmt.Errorf("EncodeInteraction: not implemented")
}
