package sui

import (
	"bytes"
	"crypto/ed25519"
	"diablo/util"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/block-vision/sui-go-sdk/models"
	"io"
	"io/ioutil"
	"net/http"
)

//const (
//	transaction_type_transfer uint8 = 0
//	//transaction_type_invoke   uint8  = 1
//	transaction_gas_limit uint64 = 2000000
//)

//type transaction interface {
//	//getTx() (*types.Transaction, error)
//}
//
//type outerTransaction struct {
//	inner virtualTransaction
//}

//func decodeTransaction(src io.Reader) (*outerTransaction, error) {
//	var inner virtualTransaction
//	var txtype uint8
//	var err error
//
//	err = util.NewMonadInputReader(src).
//		SetOrder(binary.LittleEndian).
//		ReadUint8(&txtype).
//		Error()
//	if err != nil {
//		return nil, err
//	}
//
//	switch txtype {
//	case transaction_type_transfer:
//		inner, err = decodeTransferTransaction(src)
//	default:
//		return nil, fmt.Errorf("unknown transaction type %d", txtype)
//	}
//
//	if err != nil {
//		return nil, err
//	}
//
//	return &outerTransaction{inner}, nil
//}

//type virtualTransaction interface {
//	//getTx() (virtualTransaction, *types.Transaction, error)
//}

type transferTransaction struct {
	nonce    uint64
	amount   uint64
	fromKey  ed25519.PrivateKey
	from     string
	to       string
	coinData string
	digest   string
}

func newTransferTransaction(nonce, amount uint64, fromKey ed25519.PrivateKey, from string, to string, coinData string) *transferTransaction {
	return &transferTransaction{
		nonce:    nonce,
		amount:   amount,
		fromKey:  fromKey,
		from:     from,
		to:       to,
		coinData: coinData,
		digest:   "",
	}
}

func (this *transferTransaction) encode(dest io.Writer) error {
	// TODO: check encoding, check what are encoded
	fromKey := this.fromKey.Seed()
	from := []byte(this.from)
	to := []byte(this.to)
	digest := []byte(this.digest)
	coinData := []byte(this.coinData)

	//var coinData bytes.Buffer
	//enc := gob.NewEncoder(&coinData)
	//err := enc.Encode(this.coinData)
	//if err != nil {
	//	return fmt.Errorf("error encoding coin data")
	//}

	if len(from) > 255 {
		return fmt.Errorf("from address too long (%d bytes)", len(this.from))
	}
	if len(to) > 255 {
		return fmt.Errorf("to address too long (%d bytes)", len(this.to))
	}

	return util.NewMonadOutputWriter(dest).
		SetOrder(binary.LittleEndian).
		//WriteUint8(transaction_type_transfer).
		WriteUint8(uint8(len(from))).
		WriteUint8(uint8(len(to))).
		WriteUint8(uint8(len(coinData))).
		WriteUint8(uint8(len(digest))).
		WriteUint64(this.nonce).
		WriteUint64(this.amount).
		WriteBytes(fromKey).
		WriteBytes(from).
		WriteBytes(to).
		WriteBytes(coinData).
		WriteBytes(digest).
		Error()
}

func decodeTransferTransaction(src io.Reader) (*transferTransaction, error) {
	var fromKeyBuf, fromBuf, toBuf, coinDataBuf, digestBuf []byte
	var nonce, amount uint64
	var lenFrom, lenTo, lenCoinData, lenDigest int
	// var coinDataDecodeBuf bytes.Buffer
	// var coinData []models.CoinData
	var err error

	err = util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&lenFrom).
		ReadUint8(&lenTo).
		ReadUint8(&lenCoinData).
		ReadUint8(&lenDigest).
		ReadUint64(&nonce).
		ReadUint64(&amount).
		ReadBytes(&fromKeyBuf, ed25519.SeedSize).
		ReadBytes(&fromBuf, lenFrom).
		ReadBytes(&toBuf, lenTo).
		ReadBytes(&coinDataBuf, lenCoinData).
		ReadBytes(&digestBuf, lenDigest).
		Error()
	// log.Printf("decode transfer transaction error")
	if err != nil {
		return nil, err
	}

	//dec := gob.NewDecoder(&coinDataDecodeBuf)
	//err = dec.Decode(&coinData)
	//if err != nil {
	//	return nil, err
	//}
	//log.Printf("decode coin data error")

	return newTransferTransaction(nonce, amount, ed25519.NewKeyFromSeed(fromKeyBuf), string(fromBuf), string(toBuf), string(coinDataBuf)), nil
}

// faucet requests
func RequestSuiFromFaucet(faucetHost, recipientAddress string, header map[string]string) error {

	body := models.FaucetRequest{
		FixedAmountRequest: &models.FaucetFixedAmountRequest{
			Recipient: recipientAddress,
		},
	}

	err := faucetRequest(faucetHost, body, header)

	return err
}

func faucetRequest(faucetUrl string, body interface{}, headers map[string]string) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return errors.New(fmt.Sprintf("Marshal request body error: %s", err.Error()))
	}

	req, err := http.NewRequest(http.MethodPost, faucetUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		return errors.New(fmt.Sprintf("Create request error: %s", err.Error()))
	}

	req.Header.Set("Content-Type", "application/json")

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New(fmt.Sprintf("Request faucet error: %s", err.Error()))
	}

	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New(fmt.Sprintf("Read response body error: %s", err.Error()))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return errors.New(fmt.Sprintf("Request faucet failed, statusCode: %d, err: %+v", resp.StatusCode, string(bodyBytes)))
	}

	return nil
}
