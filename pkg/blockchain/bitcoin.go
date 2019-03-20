package blockchain

import (
	"os"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"net/http"
	"path/filepath"
	"encoding/hex"
	"wallet-go/pkg/db"
	"wallet-go/pkg/util"
	"wallet-go/pkg/configure"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/manifoldco/promptui"
	"github.com/btcsuite/btcd/chaincfg"
)

var btcWalletBackupPath = strings.Join([]string{configure.Config.BackupWalletPath, "btc.backup"}, "")

// NewbitcoinClient bitcoin rpc client
func NewbitcoinClient() (*rpcclient.Client, error) {
	connCfg := &rpcclient.ConnConfig {
		Host:         configure.Config.BTCNODEHOST,
		User:         configure.Config.BTCNODEUSR,
		Pass:         configure.Config.BTCNODEPASS,
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, fmt.Errorf("Bitcoincore clienr %s", err)
	}
	return client, nil
}

// NewOmnicoreClient omnicore rpc client
func NewOmnicoreClient() (*rpcclient.Client, error) {
	connCfg := &rpcclient.ConnConfig {
		Host:         configure.Config.OmniNODEHOST,
		User:         configure.Config.OmniNODEUSR,
		Pass:         configure.Config.OmniNODEPASS,
		HTTPPostMode: configure.Config.OmniHTTPPostMode,
		DisableTLS:   configure.Config.OmniDisableTLS,
	}
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, fmt.Errorf("Omnicore clienr %s", err)
	}
	return client, nil
}

// DumpUTXO dump utxo in wallet
func (btcClient *BTCRPC) DumpUTXO() {
	resp, err := btcClient.Client.ListUnspent()
	if err != nil {
		configure.Sugar.Fatal("ListUnspent error", err.Error())
	}
	sqldb, err := db.NewMySQL()
	if err != nil {
		configure.Sugar.Fatal("NewMySQL error: ", err.Error())
	}
	defer sqldb.Close()
	ts := sqldb.Begin()
	for _, utxo := range resp {
		var subAddress db.SubAddress
		if err := ts.Where(db.SubAddress{Address: utxo.Address, Asset: "btc"}).FirstOrCreate(&subAddress).Error; err != nil {
			configure.Sugar.Fatal("insert address to db error: ", err.Error())
		}

		txHash, err := chainhash.NewHashFromStr(utxo.TxID)
		if err != nil {
			configure.Sugar.Fatal("NewHashFromStr error", err.Error())
		}
		tx, err := btcClient.Client.GetTransaction(txHash)
		if err != nil {
			configure.Sugar.Fatal("GetTransaction error", err.Error())
		}

		blockHash, err := chainhash.NewHashFromStr(tx.BlockHash)
		if err != nil {
			configure.Sugar.Fatal("NewHashFromStr error", err.Error())
		}
		rawBlock, err := btcClient.Client.GetBlockVerboseTxM(blockHash)
		if err != nil {
			configure.Sugar.Fatal("GetBlock error", err.Error())
		}

		dbUTXO := db.UTXO{Txid: utxo.TxID, Amount: utxo.Amount, Height: rawBlock.Height, VoutIndex: utxo.Vout, SubAddress: subAddress}
		dbUTXO.SetState("original")
		var qUTXO db.UTXO
		addStr := "exist"
		if err := ts.Where("txid = ? AND vout_index = ?", utxo.TxID, utxo.Vout).First(&qUTXO).Error; err != nil && strings.Contains(err.Error(), "record not found"){
			ts.Create(&dbUTXO)
			addStr = "add to db"
		}else if err != nil {
			configure.Sugar.Fatal("Fail to create utxo: ", err.Error())
		}
		configure.Sugar.Info("utxo ", "txid: ", dbUTXO.Txid, " index: ", dbUTXO.VoutIndex, " ", addStr)
	}
	if err := ts.Commit().Error; err != nil {
		if err = ts.Rollback().Error; err != nil {
			configure.Sugar.Fatal(strings.Join([]string{"database rollback error: create utxo record ", err.Error()}, ""))
		}
		configure.Sugar.Fatal(strings.Join([]string{"database commit error: ", err.Error()}, ""))
	}
}

// DumpBTC dump wallet from node
func (btcClient *BTCRPC) DumpBTC(local bool) {
	oldWalletServerClient, err := util.NewServerClient(configure.Config.OldBTCWalletServerUser,
		configure.Config.OldBTCWalletServerPass, configure.Config.OldBTCWalletServerHost)
	if err != nil {
		configure.Sugar.Fatal(err.Error())
	}
	if err = oldWalletServerClient.SftpClient.MkdirAll(filepath.Dir(configure.Config.BackupWalletPath)); err != nil {
		configure.Sugar.Fatal(err.Error())
	}

	// dump old wallet to old wallet server
	btcClient.DumpOldWallet(oldWalletServerClient)
	oldWalletServerClient.CopyRemoteFile2(btcWalletBackupPath, local)
}

// DumpOldWallet migrate old wallet from node
func (btcClient *BTCRPC) DumpOldWallet(serverClient *util.ServerClient) {
	if _, err := btcClient.Client.DumpWallet(btcWalletBackupPath); err != nil {
		if strings.Contains(err.Error(), "already exists. If you are sure this is what you want") {
			prompt := promptui.Prompt {
				Label:     strings.Join([]string{"File: ", filepath.Base(btcWalletBackupPath),
					" wallet already exists, If you are sure this is what you want, move it out of the way first "}, ""),
				IsConfirm: true,
			}
			if _, err = prompt.Run(); err != nil {
				fmt.Println("Check the old backup wallet file in", configure.Config.BackupWalletPath, "in", serverClient.SSHClient.RemoteAddr().String())
				return
			}
			if err = serverClient.SftpClient.Remove(btcWalletBackupPath); err != nil {
				configure.Sugar.Fatal("Remove backup wallet:", btcWalletBackupPath, " from old wallet server:", serverClient.SSHClient.RemoteAddr().String(), " error: ", err.Error())
			}
			btcClient.DumpOldWallet(serverClient)
		} else {
			configure.Sugar.Fatalf("DumpWallet error: ", err.Error())
		}
	} else {
		configure.Sugar.Info("dump old btc wallet result: success")
	}
}

// ImportPrivateKey import private key from dump file
func (btcClient *BTCRPC) ImportPrivateKey()  {
	file, err := os.Open(strings.Join([]string{configure.Config.BackupWalletPath, "btc.backup_new"}, ""))
	if err != nil {
		configure.Sugar.Fatal("open dump wallet file error: ", err.Error())
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "label=") && !strings.Contains(line, "script"){
			splitArr := strings.Split(line, " ")
			privateKey := splitArr[0]
			wif, err := btcutil.DecodeWIF(privateKey)
			if err != nil {
				configure.Sugar.Fatal("decode wif string error:", err.Error())
			}
			if result := btcClient.Client.ImportPrivKeyRescan(wif, "importANBI", false); result != nil {
				configure.Sugar.Fatal("fail to import private key")
			}
			configure.Sugar.Info("import success: ", strings.Split(splitArr[4], "=")[1])
		}
	}
	if err := scanner.Err(); err != nil {
		configure.Sugar.Fatal("scanner error: ", err.Error())
	}
}
// GetBlock get block with tx
func (btcClient *BTCRPC) GetBlock(height int64) (*btcjson.GetBlockVerboseResult, error) {
	blockHash, err := btcClient.Client.GetBlockHash(height)
	if err != nil {
		return nil, err
	}

	block, err := btcClient.Client.GetBlockVerboseTxM(blockHash)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// SendTx broadcast signed tx
func (btcClient *BTCRPC) SendTx(signedTx string, selectedUTXOs []db.UTXO, sqldb *db.GormDB) (*string, int, error) {
	tx, err := DecodeBtcTxHex(signedTx)
	if err != nil {
		e := errors.New(strings.Join([]string{"Decode signed tx error", err.Error()}, ":"))
		return nil, http.StatusInternalServerError, e
	}

	txHash, err := btcClient.Client.SendRawTransaction(tx.MsgTx(), false)
	if err != nil {
		e := errors.New(strings.Join([]string{"Bitcoin SendRawTransaction signed tx error", err.Error()}, ":"))
		return nil, http.StatusInternalServerError, e
	}
	txid := txHash.String()
	ts := sqldb.Begin()
	for _, dbutxo := range selectedUTXOs {
		ts.Model(&dbutxo).Updates(map[string]interface{}{"used_by": txid, "state": "selected"})
	}
	if err := ts.Commit().Error; err != nil {
		e := errors.New(strings.Join([]string{"update selected utxos error", err.Error()}, ":"))
		return nil, http.StatusInternalServerError, e
	}
	return &txid, http.StatusOK, nil
}

// DecodeBtcTxHex decode bitcoin transaction's hex to rawTx
func DecodeBtcTxHex(txHex string) (*btcutil.Tx, error) {
	if txHex == "" {
		hexErr := errors.New("signtx 交易签名参数错误")
		return nil, hexErr
	}
	txByte, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, err
	}

	var msgTx wire.MsgTx
	if err := msgTx.Deserialize(bytes.NewReader(txByte)); err != nil {
		return nil, err
	}

	return btcutil.NewTx(&msgTx), nil
}

// BitcoinNet bitcoin base chain net
func BitcoinNet(bitcoinnet string) (*chaincfg.Params, error) {
  var net chaincfg.Params
  switch bitcoinnet {
  case BitcoinTestNet, "TestNet3":
    net = chaincfg.TestNet3Params
  case BitcoinRegTest, "TestNet":
    net = chaincfg.RegressionNetParams
  case BitcoinMainnet, "MainNet":
    net = chaincfg.MainNetParams
  default:
    return nil, errors.New("bitcoinmode flag only supports testnet, regtest or mainnet")
  }
  return &net, nil
}
