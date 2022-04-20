// Copyright © 2022 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ffc

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly-ethconnect/internal/errors"
	"github.com/hyperledger/firefly-transaction-manager/pkg/ffcapi"
	"github.com/hyperledger/firefly/pkg/fftypes"
	ethbinding "github.com/kaleido-io/ethbinding/pkg"
)

// blockInfoJSONPPC are the fields we parse from the JSON/RPC response
type blockInfoJSONPPC struct {
	Number     ethbinding.HexUint `json:"number"`
	Hash       ethbinding.Hash    `json:"hash"`
	ParentHash ethbinding.Hash    `json:"parentHash"`
	Timestamp  ethbinding.HexUint `json:"timestamp"`
}

func (s *ffcServer) getBlockInfoByNumber(ctx context.Context, payload []byte) (interface{}, ffcapi.ErrorReason, error) {

	var req ffcapi.GetBlockInfoByNumberRequest
	err := json.Unmarshal(payload, &req)
	if err != nil {
		return nil, ffcapi.ErrorReasonInvalidInputs, err
	}

	blockNumber := ethbinding.HexUint64(req.BlockNumber.Uint64())
	var blockInfo *blockInfoJSONPPC
	err = s.rpc.CallContext(ctx, &blockInfo, "eth_getBlockByNumber", blockNumber, false /* only the txn hashes */)
	if err != nil {
		return nil, "", err
	}
	if blockInfo == nil {
		return nil, ffcapi.ErrorReasonNotFound, errors.Errorf(errors.FFCBlockNotAvailable)
	}

	return &ffcapi.GetBlockInfoByNumberResponse{
		BlockInfo: ffcapi.BlockInfo{
			BlockNumber: fftypes.NewFFBigInt(int64(blockInfo.Number)),
			BlockHash:   blockInfo.Hash.String(),
			ParentHash:  blockInfo.ParentHash.String(),
		},
	}, "", nil

}

func (s *ffcServer) getBlockInfoByHash(ctx context.Context, payload []byte) (interface{}, ffcapi.ErrorReason, error) {

	var req ffcapi.GetBlockInfoByHashRequest
	err := json.Unmarshal(payload, &req)
	if err != nil {
		return nil, ffcapi.ErrorReasonInvalidInputs, err
	}

	var blockInfo *blockInfoJSONPPC
	err = s.rpc.CallContext(ctx, &blockInfo, "eth_getBlockByHash", req.BlockHash, false /* only the txn hashes */)
	if err != nil {
		return nil, "", err
	}
	if blockInfo == nil {
		return nil, ffcapi.ErrorReasonNotFound, errors.Errorf(errors.FFCBlockNotAvailable)
	}

	return &ffcapi.GetBlockInfoByHashResponse{
		BlockInfo: ffcapi.BlockInfo{
			BlockNumber: fftypes.NewFFBigInt(int64(blockInfo.Number)),
			BlockHash:   blockInfo.Hash.String(),
			ParentHash:  blockInfo.ParentHash.String(),
		},
	}, "", nil

}
