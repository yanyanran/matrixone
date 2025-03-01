// Copyright 2021 - 2022 Matrix Origin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rpchandle

import (
	apipb "github.com/matrixorigin/matrixone/pkg/pb/api"
	"github.com/matrixorigin/matrixone/pkg/pb/timestamp"
	"github.com/matrixorigin/matrixone/pkg/pb/txn"
)

type Handler interface {
	HandleCommit(
		meta txn.TxnMeta,
	) error

	HandleRollback(
		meta txn.TxnMeta,
	) error

	HandleCommitting(
		meta txn.TxnMeta,
	) error

	HandlePrepare(
		meta txn.TxnMeta,
	) (
		timestamp.Timestamp,
		error,
	)

	HandleStartRecovery(
		ch chan txn.TxnMeta,
	)

	HandleClose() error

	HandleDestroy() error

	HandleGetLogTail(
		meta txn.TxnMeta,
		req apipb.SyncLogTailReq,
		resp *apipb.SyncLogTailResp,
	) error

	HandlePreCommit(
		meta txn.TxnMeta,
		req apipb.PrecommitWriteCmd,
		resp *apipb.SyncLogTailResp,
	) error
}
