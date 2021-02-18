package tron

import (
	"encoding/hex"
	"sync"

	"github.com/trustwallet/golibs/types"
)

func (p *Platform) CurrentBlockNumber() (int64, error) {
	return p.client.fetchCurrentBlockNumber()
}

func (p *Platform) GetBlockByNumber(num int64) (*types.Block, error) {
	block, err := p.client.fetchBlockByNumber(num)
	if err != nil {
		return nil, err
	}

	txsChan := p.NormalizeBlockTxs(block.Txs)
	txs := make(types.Txs, 0)
	for cTxs := range txsChan {
		txs = append(txs, cTxs)
	}

	return &types.Block{
		Number: num,
		Txs:    txs,
	}, nil
}

func (p *Platform) NormalizeBlockTxs(srcTxs []Tx) chan types.Tx {
	txChan := make(chan types.Tx, len(srcTxs))
	var wg sync.WaitGroup
	for _, srcTx := range srcTxs {
		wg.Add(1)
		go func(s Tx, c chan types.Tx) {
			defer wg.Done()
			p.NormalizeBlockChannel(s, c)
		}(srcTx, txChan)
	}
	wg.Wait()
	close(txChan)
	return txChan
}

func (p *Platform) NormalizeBlockChannel(srcTx Tx, txChan chan types.Tx) {
	if len(srcTx.Data.Contracts) == 0 {
		return
	}

	tx, err := normalize(srcTx)
	if err != nil {
		return
	}
	transfer := srcTx.Data.Contracts[0].Parameter.Value
	if len(transfer.AssetName) > 0 {
		assetName, err := hex.DecodeString(transfer.AssetName[:])
		if err == nil {
			info, err := p.gridClient.fetchTokenInfo(string(assetName))
			if err == nil && len(info.Data) > 0 {
				addTokenMeta(tx, srcTx, info.Data[0])
			}
		}
	}
	txChan <- *tx
}
