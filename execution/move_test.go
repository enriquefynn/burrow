package execution

import (
	"fmt"
	"testing"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/execution/errors"
	"github.com/hyperledger/burrow/execution/evm/abi"
	"github.com/hyperledger/burrow/execution/state"
	"github.com/hyperledger/burrow/permission"
	"github.com/hyperledger/burrow/txs"
	"github.com/hyperledger/burrow/txs/payload"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tendermint/libs/db"
	hex "github.com/tmthrgd/go-hex"
)

func TestMove2(t *testing.T) {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})

	stateDB := dbm.NewMemDB()
	defer stateDB.Close()
	genDoc := newBaseGenDoc(permission.AllAccountPermissions, permission.AllAccountPermissions)
	st, err := state.MakeGenesisState(stateDB, &genDoc)
	require.NoError(t, err)
	err = st.InitialCommit()
	require.NoError(t, err)
	exe := makeExecutor(st, testGenesisDoc)

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
	code := hex.MustDecodeString("608060405234801561001057600080fd5b50600a600081905550610141806100286000396000f3fe608060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806335169adf1461005c5780636d4ce63c146100975780636e9410b6146100c2575b600080fd5b34801561006857600080fd5b506100956004803603602081101561007f57600080fd5b81019080803590602001909291905050506100fd565b005b3480156100a357600080fd5b506100ac610107565b6040518082815260200191505060405180910390f35b3480156100ce57600080fd5b506100fb600480360360208110156100e557600080fd5b8101908080359060200190929190505050610110565b005b8060008190555050565b60008054905090565b80f65056fea165627a7a72305820d2640236c9372d97f88aafd85f26580584437b6469a57bed0e480073554d0dc60029")

	tx, _ := payload.NewCallTx(exe.stateCache, users[0].GetPublicKey(), nil, code, 1, 100000, 1)
	err = exe.signExecuteCommit(tx, users[0])

	contractAddr := crypto.NewContractAddress(tx.Input.Address, txHash(tx))
	contractAcc := getAccount(st, contractAddr)
	// contractAcc := getAccount(exe.stateCache, contractAddr)
	require.Equal(t, uint64(1), contractAcc.ShardID)

	input := hex.MustDecodeString(hex.EncodeToString(abi.GetFunctionID("m(uint256)").Bytes()))
	shardIDToMove := binary.Int64ToWord256(2)
	input = append(input, shardIDToMove.Bytes()...)
	sequence := uint64(2)

	tx = &payload.CallTx{
		Input: &payload.TxInput{
			Address:  users[0].GetAddress(),
			Amount:   0,
			Sequence: sequence,
		},
		Address:  &contractAddr,
		GasLimit: 100000000,
		Data:     input,
	}
	sequence++
	fmt.Printf("State hash: %x version %v\n", st.Hash(), st.Version())
	err = exe.signExecuteCommit(tx, users[0])
	fmt.Printf("State hash: %x version %v\n", st.Hash(), st.Version())
	require.NoError(t, err)
	contractAcc = getAccount(exe.stateCache, contractAddr)

	// Shard id should be moved to 2
	require.Equal(t, uint64(2), contractAcc.ShardID)
	logrus.Info("Contract moved to shard 2")

	// Try to call on a moved contract
	input = hex.MustDecodeString(hex.EncodeToString(abi.GetFunctionID("get()").Bytes()))
	tx = &payload.CallTx{
		Input: &payload.TxInput{
			Address:  users[0].GetAddress(),
			Amount:   0,
			Sequence: sequence,
		},
		Address:  &contractAddr,
		GasLimit: 100000000,
		Data:     input,
	}
	sequence++
	err = exe.signExecuteCommit(tx, users[0])
	// Should not be allowed to interact with contract
	require.Error(t, errors.ErrorCodeWrongShardExecution, err)

	// Get proof for account
	logrus.Infof("getting proof for contract: %v\n", contractAcc.Address)

	accountProof, err := st.GetKeyWithProof(st.GetKeyFormat().Account.Key(), contractAcc.Address)
	require.NoError(t, err)

	decodedAccount, err := acm.Decode(accountProof.DataValue)
	decodedAccount.Encode()
	isCorrect := accountProof.Verify()
	require.Equal(t, isCorrect, nil)

	// Get proof for storage, need to verify only the commit
	storageProof, err := st.GetKeyWithProof(st.GetKeyFormat().Storage.Key(), contractAcc.Address)
	require.NoError(t, err)

	storageKeyValues := binary.Int64ToWord256(0).Bytes()
	storageKeyValues = append(storageKeyValues, binary.Int64ToWord256(10).Bytes()...)

	var keys binary.Words256
	var values binary.Words256
	// Assume ordered
	for i := 0; i < len(storageKeyValues); i += 64 {
		keys = append(keys, binary.RightPadWord256(storageKeyValues[i:i+32]))
		values = append(values, binary.RightPadWord256(storageKeyValues[i+32:i+64]))

	}

	isCorrect = storageProof.VerifyStorageRoot(keys, values)
	require.Equal(t, isCorrect, nil)

	stateDB2 := dbm.NewMemDB()
	defer stateDB2.Close()
	var genesisShard2, privAccountsShard2, _ = deterministicGenesis.
		GenesisDoc(3, true, 1000, 1, true, 1000, 2)
	st2, err := state.MakeGenesisState(stateDB2, genesisShard2)
	require.NoError(t, err)
	err = st2.InitialCommit()
	require.NoError(t, err)
	exe2 := makeExecutor(st2, genesisShard2)

	input = []byte{}

	tx = &payload.CallTx{
		Input: &payload.TxInput{
			Address:  privAccountsShard2[0].GetAddress(),
			Amount:   0,
			Sequence: 1,
		},
		Address:  nil,
		GasLimit: 100000000,
		Data:     input,
		// Move
		AccountProof:   accountProof,
		StorageProof:   storageProof,
		StorageOpCodes: storageKeyValues,
		SignedHeader:   nil,
	}

	err = exe2.signExecuteCommitInChainID(tx, "2", privAccountsShard2[0])
	require.NoError(t, err)

	contractAccount, err := st2.GetAccount(contractAddr)
	fmt.Printf("Contract account: %v\n", contractAccount)
	require.NoError(t, err)

	// require.Equal(t, isCorrect, true)
	// isCorrect = accountProof.VerifyItem(accountKeyFormat.Key(decodedAccount.Address.Bytes()), movedAccount) == nil
	// require.Equal(t, isCorrect, true)

	// // Test marsheling
	// var proof2 = new(iavl.RangeProof)
	// proofBytes := make([]byte, accountProof.Size())
	// accountProof.MarshalTo(proofBytes)
	// proof2.Unmarshal(proofBytes)

	// storageKeyValues := binary.Int64ToWord256(0).Bytes()
	// storageKeyValues = append(storageKeyValues, binary.Int64ToWord256(10).Bytes()...)

	// // a := storageHashProof.Unmarshal()

	// isCorrect = proof2.Verify(stateHash) == nil
	// require.Equal(t, isCorrect, true)

	// blk1, err := st.GetBlock(1)
	// fmt.Printf("Block: %v\n", blk1)
}

func (te *testExecutor) signExecuteCommitInChainID(tx payload.Payload, chainID string, signers ...acm.AddressableSigner) error {
	txEnv := txs.Enclose(chainID, tx)
	err := txEnv.Sign(signers...)
	if err != nil {
		return err
	}
	txe, err := te.Execute(txEnv)
	if err != nil {
		return err
	}
	if txe.Exception != nil {
		return txe.Exception
	}
	_, err = te.Commit(nil)
	return err
}
