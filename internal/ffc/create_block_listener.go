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

	"github.com/hyperledger/firefly-transaction-manager/pkg/ffcapi"
	ethbinding "github.com/kaleido-io/ethbinding/pkg"
)

func (s *ffcServer) createBlockListener(ctx context.Context, payload []byte) (interface{}, ffcapi.ErrorReason, error) {

	var req ffcapi.CreateBlockListenerRequest
	err := json.Unmarshal(payload, &req)
	if err != nil {
		return nil, ffcapi.ErrorReasonInvalidInputs, err
	}

	var listenerID ethbinding.HexBigInt
	err = s.rpc.CallContext(ctx, &listenerID, "eth_newBlockFilter")
	if err != nil {
		return nil, "", err
	}

	return &ffcapi.CreateBlockListenerResponse{
		ListenerID: listenerID.String(),
	}, "", nil

}
