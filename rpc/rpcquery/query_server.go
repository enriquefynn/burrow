package rpcquery

import (
	"context"
	"fmt"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/acm/acmstate"
	"github.com/hyperledger/burrow/acm/validator"
	"github.com/hyperledger/burrow/bcm"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/consensus/tendermint"
	"github.com/hyperledger/burrow/event"
	"github.com/hyperledger/burrow/event/query"
	"github.com/hyperledger/burrow/execution/exec"
	"github.com/hyperledger/burrow/execution/names"
	"github.com/hyperledger/burrow/execution/proposal"
	"github.com/hyperledger/burrow/execution/state"
	"github.com/hyperledger/burrow/logging"
	"github.com/hyperledger/burrow/rpc"
	rpcevents "github.com/hyperledger/burrow/rpc/rpcevents"
	"github.com/hyperledger/burrow/txs/payload"
	"github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type queryServer struct {
	accounts     acmstate.IterableStatsReader
	nameReg      names.IterableReader
	proposalReg  proposal.IterableReader
	blockchain   bcm.BlockchainInfo
	validators   validator.History
	nodeView     *tendermint.NodeView
	logger       *logging.Logger
	subscribable event.Subscribable
}

var _ QueryServer = &queryServer{}

func NewQueryServer(state acmstate.IterableStatsReader, nameReg names.IterableReader, proposalReg proposal.IterableReader,
	blockchain bcm.BlockchainInfo, validators validator.History, nodeView *tendermint.NodeView, logger *logging.Logger,
	subscribable event.Subscribable) *queryServer {
	return &queryServer{
		accounts:     state,
		nameReg:      nameReg,
		proposalReg:  proposalReg,
		blockchain:   blockchain,
		validators:   validators,
		nodeView:     nodeView,
		logger:       logger,
		subscribable: subscribable,
	}
}

func (qs *queryServer) Status(ctx context.Context, param *StatusParam) (*rpc.ResultStatus, error) {
	return rpc.Status(qs.blockchain, qs.validators, qs.nodeView, param.BlockTimeWithin, param.BlockSeenTimeWithin)
}

// Account state

func (qs *queryServer) GetAccount(ctx context.Context, param *GetAccountParam) (*acm.Account, error) {
	acc, err := qs.accounts.GetAccount(param.Address)
	if acc == nil {
		acc = &acm.Account{}
	}
	return acc, err
}

func (qs *queryServer) GetAccountProofs(ctx context.Context, param *GetAccountParam) (*AccountProofs, error) {
	proofs, err := qs.accounts.GetAccountWithProof(param.Address)
	var storageOpCodes []byte
	qs.accounts.IterateStorage(param.Address, func(key, value binary.Word256) error {
		storageOpCodes = append(storageOpCodes, key[:]...)
		storageOpCodes = append(storageOpCodes, value[:]...)
		return nil
	})
	return &AccountProofs{AccountProof: *proofs[0], StorageProof: *proofs[1], StorageOpCodes: storageOpCodes}, err
}

func (qs *queryServer) GetStorage(ctx context.Context, param *GetStorageParam) (*StorageValue, error) {
	val, err := qs.accounts.GetStorage(param.Address, param.Key)
	return &StorageValue{Value: val}, err
}

func (qs *queryServer) ListAccounts(param *ListAccountsParam, stream Query_ListAccountsServer) error {
	qry, err := query.NewOrEmpty(param.Query)
	var streamErr error
	err = qs.accounts.IterateAccounts(func(acc *acm.Account) error {
		if qry.Matches(acc.Tagged()) {
			return stream.Send(acc)
		} else {
			return nil
		}
	})
	if err != nil {
		return err
	}
	return streamErr
}

// Names

func (qs *queryServer) GetName(ctx context.Context, param *GetNameParam) (entry *names.Entry, err error) {
	entry, err = qs.nameReg.GetName(param.Name)
	if entry == nil && err == nil {
		err = fmt.Errorf("name %s not found", param.Name)
	}
	return
}

func (qs *queryServer) ListNames(param *ListNamesParam, stream Query_ListNamesServer) error {
	qry, err := query.NewOrEmpty(param.Query)
	if err != nil {
		return err
	}
	var streamErr error
	err = qs.nameReg.IterateNames(func(entry *names.Entry) error {
		if qry.Matches(entry.Tagged()) {
			return stream.Send(entry)
		} else {
			return nil
		}
	})
	if err != nil {
		return err
	}
	return streamErr
}

// Validators

func (qs *queryServer) GetValidatorSet(ctx context.Context, param *GetValidatorSetParam) (*ValidatorSet, error) {
	set := validator.Copy(qs.validators.Validators(0))
	return &ValidatorSet{
		Set: set.Validators(),
	}, nil
}

func (qs *queryServer) GetValidatorSetHistory(ctx context.Context, param *GetValidatorSetHistoryParam) (*ValidatorSetHistory, error) {
	lookback := int(param.IncludePrevious)
	switch {
	case lookback == 0:
		lookback = 1
	case lookback < 0 || lookback > state.DefaultValidatorsWindowSize:
		lookback = state.DefaultValidatorsWindowSize
	}
	height := qs.blockchain.LastBlockHeight()
	if height < uint64(lookback) {
		lookback = int(height)
	}
	history := &ValidatorSetHistory{}
	for i := 0; i < lookback; i++ {
		set := validator.Copy(qs.validators.Validators(i))
		vs := &ValidatorSet{
			Height: height - uint64(i),
			Set:    set.Validators(),
		}
		history.History = append(history.History, vs)
	}
	return history, nil
}

// proposals

func (qs *queryServer) GetProposal(ctx context.Context, param *GetProposalParam) (proposal *payload.Ballot, err error) {
	proposal, err = qs.proposalReg.GetProposal(param.Hash)
	if proposal == nil && err == nil {
		err = fmt.Errorf("proposal %x not found", param.Hash)
	}
	return
}

func (qs *queryServer) ListProposals(param *ListProposalsParam, stream Query_ListProposalsServer) error {
	var streamErr error
	err := qs.proposalReg.IterateProposals(func(hash []byte, ballot *payload.Ballot) error {
		if param.GetProposed() == false || ballot.ProposalState == payload.Ballot_PROPOSED {
			return stream.Send(&ProposalResult{Hash: hash, Ballot: ballot})
		} else {
			return nil
		}
	})
	if err != nil {
		return err
	}
	return streamErr
}

func (qs *queryServer) GetStats(ctx context.Context, param *GetStatsParam) (*Stats, error) {
	stats := qs.accounts.GetAccountStats()

	return &Stats{
		AccountsWithCode:    stats.AccountsWithCode,
		AccountsWithoutCode: stats.AccountsWithoutCode,
	}, nil
}

func (qs *queryServer) streamSignedHeaders(ctx context.Context, blockRange *rpcevents.BlockRange,
	consumer func(*SignedHeadersResult) error) error {

	// Converts the bounds to half-open interval needed
	start, end, streaming := blockRange.Bounds(qs.blockchain.LastBlockHeight())
	qs.logger.TraceMsg("Streaming signed block headers", "start", start, "end", end, "streaming", streaming)

	// Pull blocks from state and receive the upper bound (exclusive) on the what we were able to send
	// Set this to start since it will be the start of next streaming batch (if needed)
	// TODO
	start, err := qs.iterateSignedHeaders(start, end, consumer)

	// If we are not streaming and all blocks requested were retrieved from state then we are done
	// TODO
	// if !streaming && start == end {
	// 	return err
	// }

	// Otherwise we need to begin streaming blocks as they are produced
	subID := event.GenSubID()
	// Subscribe to BlockExecution events
	out, err := qs.subscribable.Subscribe(ctx, subID, exec.QueryForBlockExecutionFromHeight(end),
		rpcevents.SubscribeBufferSize)
	if err != nil {
		return err
	}
	defer qs.subscribable.UnsubscribeAll(context.Background(), subID)

	for msg := range out {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			block := msg.(*exec.BlockExecution)
			streamEnd := block.Height

			finished := !streaming && streamEnd >= end
			if finished {
				// Truncate streamEnd to final end to get exactly the blocks we want from state
				streamEnd = end
			}
			if start < streamEnd {
				// This implies there are some blocks between the previous batchEnd (now start) and the current BlockExecution that
				// we have not emitted so we will pull them from state. This can occur if a block is emitted during/after
				// the initial streaming but before we have subscribed to block events or if we spill BlockExecutions
				// when streaming them and need to catch up
				_, err := qs.iterateSignedHeaders(start, streamEnd, consumer)
				if err != nil {
					return err
				}
			}
			if finished {
				return nil
			}
			commit := qs.nodeView.BlockStore().LoadBlockCommit(int64(block.Height - 1))
			header := qs.nodeView.BlockStore().LoadBlockMeta(int64(block.Height - 1)).Header
			signedHeader := tmtypes.SignedHeader{
				Commit: commit,
				Header: &header,
			}
			signedHeadersResult := SignedHeadersResult{
				SignedHeader: &signedHeader,
				TxExecutions: block.TxExecutions,
			}
			err = consumer(&signedHeadersResult)
			if err != nil {
				return err
			}
			// We've just streamed block so our next start marker is the next block
			start = block.Height + 1
		}
	}

	return nil
}

func (qs *queryServer) iterateSignedHeaders(start, end uint64, consumer func(*SignedHeadersResult) error) (uint64, error) {
	var streamErr error

	for height := start + 1; height < end-1; height++ {
		commit := qs.nodeView.BlockStore().LoadBlockCommit(int64(height))
		header := qs.nodeView.BlockStore().LoadBlockMeta(int64(height)).Header
		signedHeader := tmtypes.SignedHeader{
			Commit: commit,
			Header: &header,
		}
		signedHeadersResult := SignedHeadersResult{
			SignedHeader: &signedHeader,
		}
		streamErr = consumer(&signedHeadersResult)
	}

	if streamErr != nil {
		return 0, streamErr
	}
	// Returns the appropriate starting block for the next stream
	return end + 1, nil
}

func (qs *queryServer) ListSignedHeaders(request *rpcevents.BlocksRequest, stream Query_ListSignedHeadersServer) error {
	return qs.streamSignedHeaders(stream.Context(), request.BlockRange, func(signedHeaders *SignedHeadersResult) error {
		if signedHeaders != nil {
			// fmt.Printf("Signed HEADER: %v", signedHeader.Commit)
			err := stream.SendMsg(signedHeaders)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Tendermint and blocks

func (qs *queryServer) GetBlockHeader(ctx context.Context, param *GetBlockParam) (*types.Header, error) {
	header, err := qs.blockchain.GetBlockHeader(param.Height)
	if err != nil {
		return nil, err
	}
	abciHeader := tmtypes.TM2PB.Header(header)
	return &abciHeader, nil
}
