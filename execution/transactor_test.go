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

package execution

// func TestTransactor_BroadcastTxSync(t *testing.T) {
// 	chainID := "TestChain"
// 	bc := &bcm.Blockchain{}
// 	evc := event.NewEmitter()
// 	evc.SetLogger(logging.NewNoopLogger())
// 	txCodec := txs.NewAminoCodec()
// 	privAccount := acm.GeneratePrivateAccountFromSecret("frogs")
// 	tx := &payload.CallTx{
// 		Input: &payload.TxInput{
// 			Address: privAccount.GetAddress(),
// 		},
// 		Address: &crypto.Address{1, 2, 3},
// 	}
// 	txEnv := txs.Enclose(chainID, tx)
// 	err := txEnv.Sign(privAccount)
// 	require.NoError(t, err)
// 	height := uint64(35)
// 	trans := NewTransactor(bc, evc, NewAccounts(acmstate.NewMemoryState(), mock.NewKeyClient(privAccount), 100),
// 		func(tx tmTypes.Tx, cb func(*abciTypes.Response)) error {
// 			txe := exec.NewTxExecution(txEnv)
// 			txe.Height = height
// 			err := evc.Publish(context.Background(), txe, txe.Tagged())
// 			if err != nil {
// 				return err
// 			}
// 			bs, err := txe.Receipt.Encode()
// 			if err != nil {
// 				return err
// 			}
// 			cb(abciTypes.ToResponseCheckTx(abciTypes.ResponseCheckTx{
// 				Code: codes.TxExecutionSuccessCode,
// 				Data: bs,
// 			}))
// 			return nil
// 		}, txCodec, logger)
// 	txe, err := trans.BroadcastTxSync(context.Background(), txEnv)
// 	require.NoError(t, err)
// 	assert.Equal(t, height, txe.Height)
// }
