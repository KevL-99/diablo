package sui

import (
	"crypto/ed25519"
	"diablo/core"
	"fmt"
	"github.com/block-vision/sui-go-sdk/signer"
	"github.com/block-vision/sui-go-sdk/sui"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

type BlockchainInterface struct {
}

func (this *BlockchainInterface) Builder(params map[string]string, env []string, endpoints map[string][]string, logger core.Logger) (core.BlockchainBuilder, error) {
	var client sui.ISuiAPI
	var builder *BlockchainBuilder

	var key, value, endpoint string
	var envmap map[string][]string
	var values []string

	var err error

	logger.Debugf("new builder")

	envmap, err = parseEnvmap(env)
	if err != nil {
		return nil, err
	}

	for key = range endpoints {
		endpoint = key
		break
	}

	logger.Debugf("use endpoint '%s'", endpoint)
	client = sui.NewSuiClient("http://" + endpoint)

	builder = newBuilder(logger, client)

	for key, values = range envmap {
		if key == "accounts" {
			for _, value = range values {
				logger.Debugf("with accounts from '%s'", value)

				err = addPremadeAccounts(builder, value)
				if err != nil {
					return nil, err
				}
			}

			continue
		}
		return nil, fmt.Errorf("unknown environment key '%s'", key)
	}

	return builder, nil
}

func parseEnvmap(env []string) (map[string][]string, error) {
	var ret map[string][]string = make(map[string][]string)
	var element, key, value string
	var values []string
	var eqindex int
	var found bool

	for _, element = range env {
		eqindex = strings.Index(element, "=")
		if eqindex < 0 {
			return nil, fmt.Errorf("unexpected environment '%s'",
				element)
		}

		key = element[:eqindex]
		value = element[eqindex+1:]

		values, found = ret[key]
		if !found {
			values = make([]string, 0)
		}

		values = append(values, value)

		ret[key] = values
	}

	return ret, nil
}

type yamlAccount struct {
	Mnemonic string `yaml:"mnemonic"`
}

func addPremadeAccounts(builder *BlockchainBuilder, path string) error {
	var private ed25519.PrivateKey
	var accounts []*yamlAccount
	var address string
	var signer_acc *signer.Signer
	var decoder *yaml.Decoder
	var account *yamlAccount
	var file *os.File
	var err error

	file, err = os.Open(path)
	if err != nil {
		return err
	}

	decoder = yaml.NewDecoder(file)
	err = decoder.Decode(&accounts)

	file.Close()

	if err != nil {
		return err
	}

	for _, account = range accounts {
		signer_acc, err = signer.NewSignertWithMnemonic(account.Mnemonic)
		if err != nil {
			return err
		}

		address = signer_acc.Address
		private = signer_acc.PriKey

		builder.addAccount(address, private)
	}
	return nil
}

func (this *BlockchainInterface) Client(params map[string]string, env, view []string, logger core.Logger) (core.BlockchainClient, error) {
	var confirmer transactionConfirmer
	var preparer transactionPreparer
	var client sui.ISuiAPI
	var key, value string
	var err error

	logger.Tracef("new client")

	logger.Tracef("use endpoint '%s'", view[0])
	client = sui.NewSuiClient("http://" + view[0])

	for key, value = range params {
		if key == "prepare" {
			logger.Tracef("use prepare method '%s'", value)
			preparer, err = parsePrepare(value, logger, client)
			if err != nil {
				return nil, err
			}
			continue
		}

		return nil, fmt.Errorf("unknown parameter '%s'", key)
	}
	confirmer = newPolltxTransactionConfirmer(logger, client)

	return newClient(logger, client, preparer, confirmer), nil
}

func parsePrepare(value string, logger core.Logger, client sui.ISuiAPI) (transactionPreparer, error) {
	var preparer transactionPreparer
	var err error

	if value == "nothing" {
		if err != nil {
			return nil, err
		}
		preparer = newNothingTransactionPreparer()
		return preparer, nil
	}

	return nil, fmt.Errorf("unknown prepare method '%s'", value)
}
