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

package cnservice

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/matrixorigin/matrixone/pkg/vm/engine"

	"github.com/matrixorigin/matrixone/pkg/config"
	"github.com/matrixorigin/matrixone/pkg/fileservice"
	"github.com/matrixorigin/matrixone/pkg/frontend"
	"github.com/matrixorigin/matrixone/pkg/logservice"
	"github.com/matrixorigin/matrixone/pkg/pb/metadata"
	"github.com/matrixorigin/matrixone/pkg/taskservice"
	"github.com/matrixorigin/matrixone/pkg/txn/client"
	"github.com/matrixorigin/matrixone/pkg/txn/rpc"
	"github.com/matrixorigin/matrixone/pkg/util/toml"

	"github.com/matrixorigin/matrixone/pkg/common/morpc"
	"github.com/matrixorigin/matrixone/pkg/common/stopper"
	"go.uber.org/zap"
)

type Service interface {
	Start() error
	Close() error

	GetTaskRunner() taskservice.TaskRunner
}

type EngineType string

const (
	EngineTAE                  EngineType = "tae"
	EngineDistributedTAE       EngineType = "distributed-tae"
	EngineMemory               EngineType = "memory"
	EngineNonDistributedMemory EngineType = "non-distributed-memory"
)

// Config cn service
type Config struct {
	// UUID cn store uuid
	UUID string `toml:"uuid"`
	// Role cn node role, [AP|TP]
	Role string `toml:"role"`

	// ListenAddress listening address for receiving external requests
	ListenAddress string `toml:"listen-address"`
	// FileService file service configuration

	Engine struct {
		Type EngineType `toml:"type"`
	}

	// parameters for cn-server related buffer.
	ReadBufferSize  int
	WriteBufferSize int

	// Pipeline configuration
	Pipeline struct {
		// HostSize is the memory limit
		HostSize int64 `toml:"host-size"`
		// GuestSize is the memory limit for one query
		GuestSize int64 `toml:"guest-size"`
		// OperatorSize is the memory limit for one operator
		OperatorSize int64 `toml:"operator-size"`
		// BatchRows is the batch rows limit for one batch
		BatchRows int64 `toml:"batch-rows"`
		// BatchSize is the memory limit for one batch
		BatchSize int64 `toml:"batch-size"`
	}

	// Frontend parameters for the frontend
	Frontend config.FrontendParameters `toml:"frontend"`

	// HAKeeper configuration
	HAKeeper struct {
		// HeatbeatDuration heartbeat duration to send message to hakeeper. Default is 1s
		HeatbeatDuration toml.Duration `toml:"hakeeper-heartbeat-duration"`
		// HeatbeatTimeout heartbeat request timeout. Default is 500ms
		HeatbeatTimeout toml.Duration `toml:"hakeeper-heartbeat-timeout"`
		// DiscoveryTimeout discovery HAKeeper service timeout. Default is 30s
		DiscoveryTimeout toml.Duration `toml:"hakeeper-discovery-timeout"`
		// ClientConfig hakeeper client configuration
		ClientConfig logservice.HAKeeperClientConfig
	}

	// TaskRunner configuration
	TaskRunner struct {
		QueryLimit        int           `toml:"task-query-limit"`
		Parallelism       int           `toml:"task-parallelism"`
		MaxWaitTasks      int           `toml:"task-max-wait-tasks"`
		FetchInterval     toml.Duration `toml:"task-fetch-interval"`
		FetchTimeout      toml.Duration `toml:"task-fetch-timeout"`
		RetryInterval     toml.Duration `toml:"task-retry-interval"`
		HeartbeatInterval toml.Duration `toml:"task-heartbeat-interval"`
	}

	// RPC rpc config used to build txn sender
	RPC rpc.Config `toml:"rpc"`
}

func (c *Config) Validate() error {
	if c.UUID == "" {
		panic("missing cn store UUID")
	}
	if c.Role == "" {
		c.Role = metadata.CNRole_TP.String()
	}
	if c.HAKeeper.DiscoveryTimeout.Duration == 0 {
		c.HAKeeper.DiscoveryTimeout.Duration = time.Second * 30
	}
	if c.HAKeeper.HeatbeatDuration.Duration == 0 {
		c.HAKeeper.HeatbeatDuration.Duration = time.Second
	}
	if c.HAKeeper.HeatbeatTimeout.Duration == 0 {
		c.HAKeeper.HeatbeatTimeout.Duration = time.Millisecond * 500
	}
	if c.TaskRunner.Parallelism == 0 {
		c.TaskRunner.Parallelism = runtime.NumCPU() / 16
		if c.TaskRunner.Parallelism == 0 {
			c.TaskRunner.Parallelism = 1
		}
	}
	if c.TaskRunner.FetchInterval.Duration == 0 {
		c.TaskRunner.FetchInterval.Duration = time.Second * 10
	}
	if c.TaskRunner.FetchTimeout.Duration == 0 {
		c.TaskRunner.FetchTimeout.Duration = time.Second * 5
	}
	if c.TaskRunner.HeartbeatInterval.Duration == 0 {
		c.TaskRunner.HeartbeatInterval.Duration = time.Second * 5
	}
	if c.TaskRunner.MaxWaitTasks == 0 {
		c.TaskRunner.MaxWaitTasks = 256
	}
	if c.TaskRunner.QueryLimit == 0 {
		c.TaskRunner.QueryLimit = c.TaskRunner.Parallelism
	}
	if c.TaskRunner.RetryInterval.Duration == 0 {
		c.TaskRunner.RetryInterval.Duration = time.Second
	}
	return nil
}

type service struct {
	metadata       metadata.CNStore
	cfg            *Config
	responsePool   *sync.Pool
	logger         *zap.Logger
	server         morpc.RPCServer
	requestHandler func(ctx context.Context, message morpc.Message, cs morpc.ClientSession, engine engine.Engine, fService fileservice.FileService, cli client.TxnClient,
		messageAcquirer func() morpc.Message) error
	cancelMoServerFunc     context.CancelFunc
	mo                     *frontend.MOServer
	initHakeeperClientOnce sync.Once
	_hakeeperClient        logservice.CNHAKeeperClient
	initTxnSenderOnce      sync.Once
	_txnSender             rpc.TxnSender
	initTxnClientOnce      sync.Once
	_txnClient             client.TxnClient
	storeEngine            engine.Engine
	metadataFS             fileservice.ReplaceableFileService
	fileService            fileservice.FileService
	stopper                *stopper.Stopper

	taskService taskservice.TaskService
	taskRunner  taskservice.TaskRunner
}
