/*
 * Copyright (C) 2018 The DNA Authors
 * This file is part of The DNA library.
 *
 * The DNA is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The DNA is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The DNA.  If not, see <http://www.gnu.org/licenses/>.
 */

package txnpool

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dnaproject2/DNA/common/config"
	"github.com/dnaproject2/DNA/common/log"
	"github.com/dnaproject2/DNA/core/genesis"
	"github.com/dnaproject2/DNA/core/ledger"
	"github.com/dnaproject2/DNA/core/payload"
	"github.com/dnaproject2/DNA/core/types"
	tc "github.com/dnaproject2/DNA/txnpool/common"
	tp "github.com/dnaproject2/DNA/txnpool/proc"
	"github.com/dnaproject2/DNA/validator/stateful"
	"github.com/dnaproject2/DNA/validator/stateless"
	"github.com/ontio/ontology-eventbus/actor"
)

func TestMain(m *testing.M) {
	log.InitLog(log.InfoLog, log.Stdout)

	var err error
	ledger.DefLedger, err = ledger.NewLedger(config.DEFAULT_DATA_DIR, 0)
	if err != nil {
		log.Errorf("failed  to new ledger")
		return
	}

	m.Run()

	// tear down
	ledger.DefLedger.Close()
	os.RemoveAll(config.DEFAULT_DATA_DIR)
}

func initTestTx() *types.Transaction {
	log.InitLog(log.InfoLog, log.Stdout)
	//topic := "TXN"

	mutable := &types.MutableTransaction{
		TxType:  types.Invoke,
		Nonce:   uint32(time.Now().Unix()),
		Payload: &payload.InvokeCode{Code: []byte{}},
	}

	tx, _ := mutable.IntoImmutable()
	return tx
}

func startActor(obj interface{}) *actor.PID {
	props := actor.FromProducer(func() actor.Actor {
		return obj.(actor.Actor)
	})

	pid := actor.Spawn(props)
	return pid
}

func Test_RCV(t *testing.T) {
	var s *tp.TXPoolServer
	var wg sync.WaitGroup

	bookKeepers, err := config.DefConfig.GetBookkeepers()
	if err != nil {
		t.Error("failed to get bookkeepers")
		return
	}
	genesisConfig := config.DefConfig.Genesis
	genesisBlock, err := genesis.BuildGenesisBlock(bookKeepers, genesisConfig)
	if err != nil {
		t.Error("failed to build genesis block", err)
		return
	}
	err = ledger.DefLedger.Init(bookKeepers, genesisBlock)
	if err != nil {
		t.Error("failed to initialize default ledger", err)
		return
	}

	// Start txnpool server to receive msgs from p2p, consensus and valdiators
	s = tp.NewTxPoolServer(tc.MAX_WORKER_NUM, true, false)

	// Initialize an actor to handle the msgs from valdiators
	rspActor := tp.NewVerifyRspActor(s)
	rspPid := startActor(rspActor)
	if rspPid == nil {
		t.Error("Fail to start verify rsp actor")
		return
	}
	s.RegisterActor(tc.VerifyRspActor, rspPid)

	// Initialize an actor to handle the msgs from consensus
	tpa := tp.NewTxPoolActor(s)
	txPoolPid := startActor(tpa)
	if txPoolPid == nil {
		t.Error("Fail to start txnpool actor")
		return
	}
	s.RegisterActor(tc.TxPoolActor, txPoolPid)

	// Initialize an actor to handle the msgs from p2p and api
	ta := tp.NewTxActor(s)
	txPid := startActor(ta)
	if txPid == nil {
		t.Error("Fail to start txn actor")
		return
	}
	s.RegisterActor(tc.TxActor, txPid)

	// Start stateless validator
	statelessV, err := stateless.NewValidator("stateless")
	if err != nil {
		t.Errorf("failed to new stateless valdiator", err)
		return
	}
	statelessV.Register(rspPid)

	statelessV2, err := stateless.NewValidator("stateless2")
	if err != nil {
		t.Errorf("failed to new stateless valdiator", err)
		return
	}
	statelessV2.Register(rspPid)

	statelessV3, err := stateless.NewValidator("stateless3")
	if err != nil {
		t.Errorf("failed to new stateless valdiator", err)
		return
	}
	statelessV3.Register(rspPid)

	statefulV, err := stateful.NewValidator("stateful")
	if err != nil {
		t.Errorf("failed to new stateful valdiator", err)
		return
	}
	statefulV.Register(rspPid)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			var j int
			defer wg.Done()

			tx := initTestTx()
			for {
				j++
				txReq := &tc.TxReq{
					Tx:     tx,
					Sender: tc.NilSender,
				}
				txPid.Tell(txReq)

				if j >= 4 {
					return
				}
			}
		}()
	}

	wg.Wait()
	time.Sleep(1 * time.Second)
	txPoolPid.Tell(&tc.GetTxnPoolReq{ByCount: true})
	txPoolPid.Tell(&tc.GetPendingTxnReq{ByCount: true})
	time.Sleep(2 * time.Second)

	statelessV.UnRegister(rspPid)
	statelessV2.UnRegister(rspPid)
	statelessV3.UnRegister(rspPid)
	statefulV.UnRegister(rspPid)
	s.Stop()
}