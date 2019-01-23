// Copyright 2017 Monax Industries Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package state

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/crypto/sha3"
	"github.com/hyperledger/burrow/execution/errors"
	"github.com/hyperledger/burrow/execution/evm/abi"
	"github.com/hyperledger/burrow/execution/exec"
	"github.com/hyperledger/burrow/permission"
	"github.com/hyperledger/burrow/txs"
	"github.com/hyperledger/burrow/txs/payload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

func TestState_UpdateAccount(t *testing.T) {
	s := NewState(dbm.NewMemDB())
	account := acm.NewAccountFromSecret("Foo")
	account.Permissions.Base.Perms = permission.SetGlobal | permission.HasRole
	_, _, err := s.Update(func(ws Updatable) error {
		return ws.UpdateAccount(account)
	})
	require.NoError(t, err)

	require.NoError(t, err)
	accountOut, err := s.GetAccount(account.Address)
	require.NoError(t, err)
	assert.Equal(t, account, accountOut)
}

func TestWriteState_AddBlock(t *testing.T) {
	s := NewState(dbm.NewMemDB())
	height := uint64(100)
	numTxs := uint64(5)
	events := uint64(10)
	block := mkBlock(height, numTxs, events)
	_, _, err := s.Update(func(ws Updatable) error {
		return ws.AddBlock(block)
	})
	require.NoError(t, err)
	ti := uint64(0)
	err = s.IterateStreamEvents(height, height+1,
		func(ev *exec.StreamEvent) error {
			if ev.TxExecution != nil {
				for e := uint64(0); e < events; e++ {
					require.Equal(t, mkEvent(height, ti, e).Header.TxHash.String(),
						ev.TxExecution.Events[e].Header.TxHash.String(), "event TxHash mismatch at tx #%d event #%d", ti, e)
				}
				ti++
			}

			return nil
		})
	require.NoError(t, err)
	// non-increasing events
	_, _, err = s.Update(func(ws Updatable) error {
		return nil
	})
	require.NoError(t, err)

	txExecutions, err := s.TxsAtHeight(height)
	require.NoError(t, err)
	require.NotNil(t, txExecutions)
	require.Equal(t, numTxs, uint64(len(txExecutions)))
}

func newParams(shardID uint64) evm.Params {
	return evm.Params{
		BlockHeight: 0,
		BlockHash:   binary.Zero256,
		BlockTime:   0,
		GasLimit:    0,
		ShardID:     shardID,
	}
}

func Test1(t *testing.T) {
	var shardID1 uint64 = 1
	chainID1 := strconv.Itoa(int(shardID1))

	st, privAccounts := makeGenesisState(3, true, 1000, 1, true, 1000, shardID1)

	acc0 := getAccount(st, privAccounts[0].GetAddress())
	acc0.Balance = 10000000

	/*
		pragma solidity >0.4.25;
		contract Example {
			int v;
			constructor() public {
				v = 10;
			}
			function setV(int _v) public {
				v = _v;
			}
			function m(uint shardID) public {
				assembly {
					move(shardID)
				}
			}
			function get() public view returns(int) {
				return v;
			}
		}
	*/
	code := hex.MustDecodeString("608060405234801561001057600080fd5b50600a600081905550610141806100286000396000f3fe608060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806335169adf1461005c5780636d4ce63c146100975780636e9410b6146100c2575b600080fd5b34801561006857600080fd5b506100956004803603602081101561007f57600080fd5b81019080803590602001909291905050506100fd565b005b3480156100a357600080fd5b506100ac610107565b6040518082815260200191505060405180910390f35b3480156100ce57600080fd5b506100fb600480360360208110156100e557600080fd5b8101908080359060200190929190505050610110565b005b8060008190555050565b60008054905090565b80f65056fea165627a7a72305820654155e63b99471edec7c188c27f2df5250152a6f6872367c5cad6e8ba16f1220029")

	_, err := st.Update(func(up Updatable) error {
		return up.UpdateAccount(acc0)
	})
	require.NoError(t, err)

	// m(uint256)
	input := hex.MustDecodeString(hex.EncodeToString(abi.GetFunctionID("m(uint256)").Bytes()))
	shardIDToMove := binary.Int64ToWord256(2)
	input = append(input, shardIDToMove.Bytes()...)

	exe := makeExecutorWithGenesis(st, testGenesisDoc)

	tx, _ := payload.NewCallTx(exe.stateCache, privAccounts[0].GetPublicKey(), nil, code, 1, 100000, 1)
	err = exe.signExecuteCommit(tx, chainID1, privAccounts[0])

	require.NoError(t, err)
	contractAddr := crypto.NewContractAddress(tx.Input.Address, txHash(tx))
	contractAcc := getAccount(exe.stateCache, contractAddr)
	// Shard id should be 1
	require.Equal(t, shardID1, contractAcc.ShardID)

	_, err = st.Update(func(up Updatable) error {
		return up.UpdateAccount(acc0)
	})
	// fmt.Printf("Account: %v %v\n", contractAcc.Address, contractAcc.ShardID)

	tx = &payload.CallTx{
		Input: &payload.TxInput{
			Address:  acc0.Address,
			Amount:   0,
			Sequence: acc0.Sequence + 1,
		},
		Address:  &contractAddr,
		GasLimit: 100000000,
		Data:     input,
	}
	err = exe.signExecuteCommit(tx, chainID1, privAccounts[0])
	require.NoError(t, err)
	_, err = st.Update(func(up Updatable) error {
		return up.UpdateAccount(acc0)
	})
}

func TestMove2(t *testing.T) {
	var shardID1 uint64 = 1
	var shardID2 uint64 = 2
	chainID1 := strconv.Itoa(int(shardID1))
	chainID2 := strconv.Itoa(int(shardID2))

	st, privAccounts := makeGenesisState(3, true, 1000, 1, true, 1000, shardID1)
	st2, privAccounts2 := makeGenesisState(3, true, 1000, 1, true, 1000, shardID2)

	acc02 := getAccount(st2, privAccounts2[0].GetAddress())
	acc02.Balance = 10000000

	acc0 := getAccount(st, privAccounts[0].GetAddress())
	acc0.Balance = 10000000

	/*
		pragma solidity >0.4.25;
		contract Example {
			int v;
			constructor() public {
				v = 10;
			}
			function setV(int _v) public {
				v = _v;
			}
			function m(uint shardID) public {
				assembly {
					move(shardID)
				}
			}
			function get() public view returns(int) {
				return v;
			}
		}
	*/
	code := hex.MustDecodeString("608060405234801561001057600080fd5b50600a600081905550610141806100286000396000f3fe608060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806335169adf1461005c5780636d4ce63c146100975780636e9410b6146100c2575b600080fd5b34801561006857600080fd5b506100956004803603602081101561007f57600080fd5b81019080803590602001909291905050506100fd565b005b3480156100a357600080fd5b506100ac610107565b6040518082815260200191505060405180910390f35b3480156100ce57600080fd5b506100fb600480360360208110156100e557600080fd5b8101908080359060200190929190505050610110565b005b8060008190555050565b60008054905090565b80f65056fea165627a7a72305820654155e63b99471edec7c188c27f2df5250152a6f6872367c5cad6e8ba16f1220029")

	_, err := st.Update(func(up Updatable) error {
		return up.UpdateAccount(acc0)
	})
	require.NoError(t, err)
	_, err = st2.Update(func(up Updatable) error {
		return up.UpdateAccount(acc02)
	})
	require.NoError(t, err)

	// m(uint256)
	input := hex.MustDecodeString(hex.EncodeToString(abi.GetFunctionID("m(uint256)").Bytes()))
	shardIDToMove := binary.Int64ToWord256(2)
	input = append(input, shardIDToMove.Bytes()...)

	exe := makeExecutorWithGenesis(st, testGenesisDoc)

	tx, _ := payload.NewCallTx(exe.stateCache, privAccounts[0].GetPublicKey(), nil, code, 1, 100000, 1)
	err = exe.signExecuteCommit(tx, chainID1, privAccounts[0])

	require.NoError(t, err)
	contractAddr := crypto.NewContractAddress(tx.Input.Address, txHash(tx))
	contractAcc := getAccount(exe.stateCache, contractAddr)
	// Shard id should be 1
	require.Equal(t, shardID1, contractAcc.ShardID)

	_, err = st.Update(func(up Updatable) error {
		return up.UpdateAccount(acc0)
	})
	// fmt.Printf("Account: %v %v\n", contractAcc.Address, contractAcc.ShardID)

	tx = &payload.CallTx{
		Input: &payload.TxInput{
			Address:  acc0.Address,
			Amount:   0,
			Sequence: acc0.Sequence + 1,
		},
		Address:  &contractAddr,
		GasLimit: 100000000,
		Data:     input,
	}
	err = exe.signExecuteCommit(tx, chainID1, privAccounts[0])
	require.NoError(t, err)
	_, err = st.Update(func(up Updatable) error {
		return up.UpdateAccount(acc0)
	})
	contractAcc = getAccount(exe.stateCache, contractAddr)

	// Shard id should be moved to 2
	require.Equal(t, uint64(2), contractAcc.ShardID)

	input = hex.MustDecodeString(hex.EncodeToString(abi.GetFunctionID("get()").Bytes()))
	tx = &payload.CallTx{
		Input: &payload.TxInput{
			Address:  acc0.Address,
			Amount:   0,
			Sequence: acc0.Sequence + 1,
		},
		Address:  &contractAddr,
		GasLimit: 100000000,
		Data:     input,
	}
	err = exe.signExecuteCommit(tx, chainID1, privAccounts[0])
	// Should not be allowed to interact with contract
	require.Error(t, errors.ErrorCodeWrongShardExecution, err)

	// Get proof for account
	movedAccount, accountProof, err := st.GetAccountWithProof(contractAcc.Address)

	decodedAccount, err := acm.Decode(movedAccount)
	stateHash := st.Hash()

	decodedAccount.Encode()

	isCorrect := accountProof.Verify(stateHash) == nil
	require.Equal(t, isCorrect, true)
	isCorrect = accountProof.VerifyItem(accountKeyFormat.Key(decodedAccount.Address.Bytes()), movedAccount) == nil
	require.Equal(t, isCorrect, true)

	// Test marsheling
	var proof2 = new(iavl.RangeProof)
	proofBytes := make([]byte, accountProof.Size())
	accountProof.MarshalTo(proofBytes)
	proof2.Unmarshal(proofBytes)

	storageKeyValues := binary.Int64ToWord256(0).Bytes()
	storageKeyValues = append(storageKeyValues, binary.Int64ToWord256(10).Bytes()...)

	// a := storageHashProof.Unmarshal()

	isCorrect = proof2.Verify(stateHash) == nil
	require.Equal(t, isCorrect, true)

	exe2 := makeExecutorWithGenesis(st2, testGenesisDoc2)

	input = []byte{}

	tx = &payload.CallTx{
		Input: &payload.TxInput{
			Address:  acc02.Address,
			Amount:   0,
			Sequence: acc02.Sequence + 1,
		},
		Address:        nil,
		GasLimit:       100000000,
		Data:           input,
		BlockRoot:      stateHash,
		AccountProof:   accountProof,
		MovedAccount:   decodedAccount,
		StorageOpCodes: storageKeyValues,
	}
	err = exe2.signExecuteCommit(tx, chainID2, privAccounts2[0])
	require.NoError(t, err)

	// err = exe.signExecuteCommit(tx, privAccounts[0])
	// st2 := NewState(db.NewMemDB())
	// // account21 := newAccount(cache2, "1, 2, 3", 2)
	// account22 := acm.NewAccountFromSecret("foobar")

	// ourVm := evm.NewVM(newParams(2), crypto.ZeroAddress, nil, logger)
	// // var gas2 uint64 = 100000

	// var gas uint64 = 100000

	// code := hex.MustDecodeString("608060405234801561001057600080fd5b50600a600081905550610141806100286000396000f3fe608060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806335169adf1461005c5780636d4ce63c146100975780636e9410b6146100c2575b600080fd5b34801561006857600080fd5b506100956004803603602081101561007f57600080fd5b81019080803590602001909291905050506100fd565b005b3480156100a357600080fd5b506100ac610107565b6040518082815260200191505060405180910390f35b3480156100ce57600080fd5b506100fb600480360360208110156100e557600080fd5b8101908080359060200190929190505050610110565b005b8060008190555050565b60008054905090565b80f65056fea165627a7a72305820d2640236c9372d97f88aafd85f26580584437b6469a57bed0e480073554d0dc60029")

	// // Run the contract initialisation code to obtain the contract code that would be mounted at account2
	// contractCode, err := ourVm.Call(st, NewNoopEventSink(), account1, account2, code, code, 0, &gas)
	// require.NoError(t, err)

	// // Not needed for this test (since contract code is passed as argument to vm), but this is what an execution
	// // framework must do
	// cache.InitCode(account2, contractCode)

	// // m(uint256) move to shard 255
	// input := hex.MustDecodeString("6e9410b6")
	// shardID := Int64ToWord256(255)
	// input = append(input, shardID.Bytes()...)

	// _, err = ourVm.Call(cache, NewNoopEventSink(), account1, account2, contractCode, input, 0, &gas)
	// require.NoError(t, err)
	// assert.Equal(t, uint64(255), cache.GetShardID(account2))
	// cache.Sync()

	// // get() : should fail because contract cannot be called while moved
	// input = hex.MustDecodeString("6d4ce63c")
	// _, err = ourVm.Call(cache, NewNoopEventSink(), account1, account2, contractCode, input, 0, &gas)
	// require.Error(t, err)

	// originalStorage := Int64ToWord256(0).Bytes()
	// originalStorage = append(originalStorage, Int64ToWord256(10).Bytes()...)
	// st.GetAccountWithProof(account22)
	// fmt.Printf("Original storage hash for %v: %x\n", account22, cache.GetStorageHash(account22))

	// // contractCode, err = ourVm2.Move2(cache2, NewNoopEventSink(), account21, account22, code, []byte{}, originalStorage, []byte{}, 0, &gas2, 0)
	// // cache.InitCode(account22, contractCode)
	// // require.NoError(t, err)
}

func mkBlock(height, numTxs, events uint64) *exec.BlockExecution {
	be := &exec.BlockExecution{
		Height: height,
	}
	for ti := uint64(0); ti < numTxs; ti++ {
		hash := txs.NewTx(&payload.CallTx{}).Hash()
		hash[0] = byte(ti)
		txe := &exec.TxExecution{
			TxHash: hash,
			Height: height,
			Index:  ti,
		}
		for e := uint64(0); e < events; e++ {
			txe.Events = append(txe.Events, mkEvent(height, ti, e))
		}
		be.TxExecutions = append(be.TxExecutions, txe)
	}
	return be
}

func mkEvent(height, tx, index uint64) *exec.Event {
	return &exec.Event{
		Header: &exec.Header{
			Height:  height,
			Index:   index,
			TxHash:  sha3.Sha3([]byte(fmt.Sprintf("txhash%v%v%v", height, tx, index))),
			EventID: fmt.Sprintf("eventID: %v%v%v", height, tx, index),
		},
		Log: &exec.LogEvent{
			Address: crypto.Address{byte(height), byte(index)},
			Topics:  []binary.Word256{{1, 2, 3}},
		},
	}
}
