package sui

import (
	"bytes"
	"context"
	"diablo/core"
	"fmt"
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/sui"
	"log"
	"time"
)

type BlockchainClient struct {
	logger    core.Logger
	client    sui.ISuiAPI
	preparer  transactionPreparer
	confirmer transactionConfirmer
}

func newClient(logger core.Logger, client sui.ISuiAPI, preparer transactionPreparer, confirmer transactionConfirmer) *BlockchainClient {
	return &BlockchainClient{
		logger:    logger,
		client:    client,
		preparer:  preparer,
		confirmer: confirmer,
	}
}

func (this *BlockchainClient) DecodePayload(encoded []byte) (interface{}, error) {
	var buffer *bytes.Buffer = bytes.NewBuffer(encoded)
	var tx *transferTransaction
	var err error
	log.Printf("entered decode payload")
	tx, err = decodeTransferTransaction(buffer)
	if err != nil {
		return nil, err
	}

	log.Printf("transfer transaction decoded")
	this.logger.Tracef("decode transaction %p", tx)

	err = this.preparer.prepare(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (this *BlockchainClient) TriggerInteraction(iact core.Interaction) error {
	var ctx = context.Background()
	var tx = iact.Payload().(*transferTransaction)

	this.logger.Tracef("submit transfer transaction %p", tx)
	iact.ReportSubmit()

	rsp, err := this.client.TransferSui(ctx, models.TransferSuiRequest{
		Signer: tx.from,
		// TODO: always fetch [0] or check for the newest one? (or the oldest, cuz they all get funded with initial gas coin and want send from that)
		// TODO: after recv, does the recv has enough coin for gas?
		SuiObjectId: tx.coinData,
		GasBudget:   "100000000",
		Recipient:   tx.to,
		// TODO: make check amount, don't hardcode it, make it tx.amount
		Amount: "1000",
	})
	if err != nil {
		iact.ReportAbort()
		return err
	}

	this.logger.Tracef("sign and execute transfer transaction %p", tx)

	rsp2, err := this.client.SignAndExecuteTransactionBlock(ctx, models.SignAndExecuteTransactionBlockRequest{
		TxnMetaData: rsp,
		PriKey:      tx.fromKey,
		Options: models.SuiTransactionBlockOptions{
			ShowInput:    true,
			ShowRawInput: true,
			ShowEffects:  true,
		},
		RequestType: "WaitForLocalExecution",
	})

	if err != nil {
		iact.ReportAbort()
		return err
	}
	// TODO: is this the right way to set digest? Will it be set after function exits? Is tx a local var or ptr?
	iact.Payload().(*transferTransaction).digest = rsp2.Digest
	return this.confirmer.confirm(iact)
}

type transactionPreparer interface {
	prepare(*transferTransaction) error
}
type nothingTransactionPreparer struct {
}

func newNothingTransactionPreparer() transactionPreparer {
	return &nothingTransactionPreparer{}
}

func (this *nothingTransactionPreparer) prepare(*transferTransaction) error {
	return nil
}

type transactionConfirmer interface {
	confirm(core.Interaction) error
}

type polltxTransactionConfirmer struct {
	logger core.Logger
	client sui.ISuiAPI
	mwait  time.Duration
}

func newPolltxTransactionConfirmer(logger core.Logger, client sui.ISuiAPI) *polltxTransactionConfirmer {
	return &polltxTransactionConfirmer{
		logger: logger,
		client: client,
		mwait:  30 * time.Second,
	}
}

func (this *polltxTransactionConfirmer) confirm(iact core.Interaction) error {
	//TODO: this is really sketchy, check it over, does get block really mean it's committed?
	var tx *transferTransaction
	var err error
	var ctx = context.Background()
	tx = iact.Payload().(*transferTransaction)
	if tx.digest == "" {
		iact.ReportAbort()
		return fmt.Errorf("digest is empty, transaction never went through")
	}
	for i := 0; i < 3; i++ {
		_, err := this.client.SuiGetTransactionBlock(ctx, models.SuiGetTransactionBlockRequest{
			Digest: tx.digest,
			Options: models.SuiTransactionBlockOptions{
				ShowInput:    true,
				ShowRawInput: true,
				ShowEffects:  true,
			},
		})
		if err != nil {
			time.Sleep(this.mwait)
		} else {
			break
		}
	}
	if err != nil {
		iact.ReportAbort()
		this.logger.Errorf("error confirming transaction")
		return err
	}
	iact.ReportCommit()
	return nil
}
