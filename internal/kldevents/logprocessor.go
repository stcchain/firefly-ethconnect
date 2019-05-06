// Copyright 2019 Kaleido

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kldevents

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/kaleido-io/ethconnect/internal/kldeth"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/kaleido-io/ethconnect/internal/kldbind"
)

type logEntry struct {
	Address          kldbind.Address   `json:"address"`
	BlockNumber      kldbind.HexBigInt `json:"blockNumber"`
	TransactionIndex json.Number       `json:"transactionIndex"`
	TransactionHash  kldbind.Hash      `json:"transactionHash"`
	Data             string            `json:"data"`
	Topics           []*kldbind.Hash   `json:"topics"`
}

type eventData struct {
	Address          string                 `json:"address"`
	BlockNumber      string                 `json:"blockNumber"`
	TransactionIndex string                 `json:"transactionIndex"`
	TransactionHash  string                 `json:"transactionHash"`
	Data             map[string]interface{} `json:"data"`
	SubID            string                 `json:"subId"`
	// Used for callback handling
	batchComplete func(*eventData)
}

type logProcessor struct {
	subID    string
	event    *kldbind.ABIEvent
	action   *action
	blockHWM big.Int
	hwnSync  sync.Mutex
}

func newLogProcessor(subID string, event *kldbind.ABIEvent, action *action) *logProcessor {
	return &logProcessor{
		subID:  subID,
		event:  event,
		action: action,
	}
}

func (lp *logProcessor) batchComplete(newestEvent *eventData) {
	lp.hwnSync.Lock()
	i := new(big.Int)
	i.SetString(newestEvent.BlockNumber, 10)
	if i.Cmp(&lp.blockHWM) > 0 {
		lp.blockHWM.Set(i)
	}
	lp.hwnSync.Unlock()
}

func (lp *logProcessor) getBlockHWM() big.Int {
	lp.hwnSync.Lock()
	v := lp.blockHWM
	lp.hwnSync.Unlock()
	return v
}

func (lp *logProcessor) processLogEntry(entry *logEntry) (err error) {

	var data []byte
	if strings.HasPrefix(entry.Data, "0x") {
		data, err = kldbind.HexDecode(entry.Data)
		if err != nil {
			return fmt.Errorf("Failed to decode data: %s", err)
		}
	}

	result := &eventData{
		Address:          entry.Address.String(),
		BlockNumber:      entry.BlockNumber.ToInt().String(),
		TransactionIndex: entry.TransactionIndex.String(),
		TransactionHash:  entry.TransactionHash.String(),
		Data:             make(map[string]interface{}),
		SubID:            lp.subID,
		batchComplete:    lp.batchComplete,
	}
	topicIdx := 0
	if !lp.event.Anonymous {
		topicIdx++ // first index is the hash of the event description
	}

	// We need split out the indexed args that we parse out of the topic, from the data args
	var dataArgs kldbind.ABIArguments
	dataArgs = make([]kldbind.ABIArgument, 0, len(lp.event.Inputs))
	for idx, input := range lp.event.Inputs {
		var val interface{}
		if input.Indexed {
			if topicIdx >= len(entry.Topics) {
				return fmt.Errorf("Ran out of topics for indexed fields at field %d of %s", idx, lp.event)
			}
			topic := entry.Topics[topicIdx]
			topicIdx++
			if topic != nil {
				val = topicToValue(topic, &input)
			} else {
				val = nil
			}
			result.Data[input.Name] = val
		} else {
			dataArgs = append(dataArgs, input)
		}
	}

	// Retrieve the data args from the RLP and merge the results
	dataMap, err := kldeth.ProcessRLPBytes(dataArgs, data)
	if err != nil {
		return fmt.Errorf("Failed to parse RLP data from event: %s", err)
	}
	for k, v := range dataMap {
		result.Data[k] = v
	}

	// Ok, now we have the full event in a friendly map output. Pass it down to the event processor
	lp.action.HandleEvent(result)
	return nil
}

func topicToValue(topic *kldbind.Hash, input *kldbind.ABIArgument) interface{} {
	switch input.Type.T {
	case kldbind.IntTy, kldbind.UintTy, kldbind.BoolTy:
		h := kldbind.HexBigInt{}
		h.UnmarshalText([]byte(topic.Hex()))
		bI, _ := math.ParseBig256(topic.Hex())
		if input.Type.T == kldbind.IntTy {
			// It will be a two's complement number, so needs to be interpretted
			bI = math.S256(bI)
			return bI.String()
		} else if input.Type.T == kldbind.BoolTy {
			return (bI.Uint64() != 0)
		}
		return bI.String()
	case kldbind.AddressTy:
		topicBytes := topic.Bytes()
		addrBytes := topicBytes[len(topicBytes)-20:]
		return kldbind.BytesToAddress(addrBytes)
	default:
		// For all other types it is just a hash of the output for indexing, so we can only
		// logically return it as a hex string. The Solidity developer has to include
		// the same data a second type non-indexed to get the real value.
		return topic.String()
	}
}
