package rpctransact

import (
	"fmt"
	"time"

	"github.com/hyperledger/burrow/execution"
	"github.com/hyperledger/burrow/execution/exec"
	"github.com/hyperledger/burrow/txs"
	"github.com/hyperledger/burrow/txs/payload"
	"golang.org/x/net/context"
)

// This is probably silly
const maxBroadcastSyncTimeout = time.Hour

type transactServer struct {
	transactor *execution.Transactor
	txCodec    txs.Codec
}

func NewTransactServer(transactor *execution.Transactor, txCodec txs.Codec) TransactServer {
	return &transactServer{
		transactor: transactor,
		txCodec:    txCodec,
	}
}

func (ts *transactServer) BroadcastTxSync(ctx context.Context, param *TxEnvelopeParam) (*exec.TxExecution, error) {
	const errHeader = "BroadcastTxSync():"
	if param.Timeout == 0 {
		param.Timeout = maxBroadcastSyncTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, param.Timeout)
	defer cancel()
	txEnv := param.GetEnvelope(ts.transactor.BlockchainInfo.ChainID())
	if txEnv == nil {
		return nil, fmt.Errorf("%s no transaction envelope or payload provided", errHeader)
	}
	return ts.transactor.BroadcastTxSync(ctx, txEnv)
}

func (ts *transactServer) BroadcastTxAsync(ctx context.Context, param *TxEnvelopeParam) (*txs.Receipt, error) {
	const errHeader = "BroadcastTxAsync():"
	if param.Timeout == 0 {
		param.Timeout = maxBroadcastSyncTimeout
	}
	txEnv := param.GetEnvelope(ts.transactor.BlockchainInfo.ChainID())
	if txEnv == nil {
		return nil, fmt.Errorf("%s no transaction envelope or payload provided", errHeader)
	}
	return ts.transactor.BroadcastTxAsync(ctx, txEnv)
}

func (ts *transactServer) BroadcastBatchAsync(ctx context.Context, param *TxEnvelopeBatchParam) (*ReceiptBatch, error) {
	receipts := &ReceiptBatch{}
	if param.Timeout == 0 {
		param.Timeout = maxBroadcastSyncTimeout
	}

	for _, txEnv := range param.Envelope {
		const errHeader = "BroadcastBatchAsync():"
		if param.Timeout == 0 {
			param.Timeout = maxBroadcastSyncTimeout
		}
		receipt, err := ts.transactor.BroadcastTxAsync(ctx, txEnv)
		if err != nil {
			return nil, err
		}
		receipts.Receipts = append(receipts.Receipts, receipt)
	}

	// for _, txEnv := range param.Envelope {
	// 	r, err := ts.transactor.BroadcastTxAsync(ctx, &txEnv)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	receipt.Receipts = append(receipt.Receipts, r)
	// }
	return receipts, nil
}

func (ts *transactServer) SignTx(ctx context.Context, param *TxEnvelopeParam) (*TxEnvelope, error) {
	txEnv := param.GetEnvelope(ts.transactor.BlockchainInfo.ChainID())
	if txEnv == nil {
		return nil, fmt.Errorf("no transaction envelope or payload provided")
	}
	txEnv, err := ts.transactor.SignTx(txEnv)
	if err != nil {
		return nil, err
	}
	return &TxEnvelope{
		Envelope: txEnv,
	}, nil
}

func (ts *transactServer) FormulateTx(ctx context.Context, param *payload.Any) (*TxEnvelope, error) {
	txEnv := txs.EnvelopeFromAny(ts.transactor.BlockchainInfo.ChainID(), param)
	if txEnv == nil {
		return nil, fmt.Errorf("no payload provided to FormulateTx")
	}
	return &TxEnvelope{
		Envelope: txEnv,
	}, nil
}

func (ts *transactServer) CallTxSync(ctx context.Context, param *payload.CallTx) (*exec.TxExecution, error) {
	return ts.BroadcastTxSync(ctx, &TxEnvelopeParam{Payload: param.Any()})
}

func (ts *transactServer) CallTxAsync(ctx context.Context, param *payload.CallTx) (*txs.Receipt, error) {
	return ts.BroadcastTxAsync(ctx, &TxEnvelopeParam{Payload: param.Any()})
}

func (ts *transactServer) CallTxSim(ctx context.Context, param *payload.CallTx) (*exec.TxExecution, error) {
	if param.Address == nil {
		return nil, fmt.Errorf("CallSim requires a non-nil address from which to retrieve code")
	}
	return ts.transactor.CallSim(param.Input.Address, *param.Address, param.Data)
}

func (ts *transactServer) CallCodeSim(ctx context.Context, param *CallCodeParam) (*exec.TxExecution, error) {
	return ts.transactor.CallCodeSim(param.FromAddress, param.Code, param.Data)
}

func (ts *transactServer) SendTxSync(ctx context.Context, param *payload.SendTx) (*exec.TxExecution, error) {
	return ts.BroadcastTxSync(ctx, &TxEnvelopeParam{Payload: param.Any()})
}

func (ts *transactServer) SendTxAsync(ctx context.Context, param *payload.SendTx) (*txs.Receipt, error) {
	return ts.BroadcastTxAsync(ctx, &TxEnvelopeParam{Payload: param.Any()})
}

func (ts *transactServer) NameTxSync(ctx context.Context, param *payload.NameTx) (*exec.TxExecution, error) {
	return ts.BroadcastTxSync(ctx, &TxEnvelopeParam{Payload: param.Any()})
}

func (ts *transactServer) NameTxAsync(ctx context.Context, param *payload.NameTx) (*txs.Receipt, error) {
	return ts.BroadcastTxAsync(ctx, &TxEnvelopeParam{Payload: param.Any()})
}

func (te *TxEnvelopeParam) GetEnvelope(chainID string) *txs.Envelope {
	if te == nil {
		return nil
	}
	if te.Envelope != nil {
		return te.Envelope
	}
	if te.Payload != nil {
		return txs.EnvelopeFromAny(chainID, te.Payload)
	}
	return nil
}
