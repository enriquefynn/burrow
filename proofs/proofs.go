package proofs

import (
	"fmt"
	"reflect"

	"github.com/hyperledger/burrow/storage"
	amino "github.com/tendermint/go-amino"
	"github.com/tendermint/iavl"
)

type ShardProof struct {
	AccountProof *Proof
	StorageProof *Proof
}

func NewShardProof(accountProof, storageProof *Proof) *ShardProof {
	return &ShardProof{
		AccountProof: accountProof,
		StorageProof: storageProof,
	}
}

type Proof struct {
	CommitProof *iavl.RangeProof
	DataProof   *iavl.RangeProof
	CommitValue []byte
	DataValues  [][]byte
	DataKeys    [][]byte
}

func NewProof(commitProof, dataProof *iavl.RangeProof, commitValue []byte, dataKeys, dataValues [][]byte) *Proof {
	return &Proof{
		CommitProof: commitProof,
		DataProof:   dataProof,
		CommitValue: commitValue,
		DataKeys:    dataKeys,
		DataValues:  dataValues,
	}
}

func (proof *Proof) Unmarshal(tree []byte) error {
	cdc := amino.NewCodec()
	return cdc.UnmarshalBinaryBare(tree, proof)
}

func (proof *Proof) MarshalTo(tree []byte) (int, error) {
	cdc := amino.NewCodec()
	proofBytes, err := cdc.MarshalBinaryBare(proof)
	if err != nil {
		return 0, err
	}
	return copy(tree, proofBytes), nil
}

func (proof *Proof) Size() int {
	cdc := amino.NewCodec()
	proofBytes := cdc.MustMarshalBinaryBare(proof)
	return len(proofBytes)
}

func (p *Proof) Verify() error {
	blockRoot := p.CommitProof.ComputeRootHash()
	dataRoot := p.DataProof.ComputeRootHash()
	// Verify Commit proof
	err := p.CommitProof.Verify(blockRoot)
	if err != nil {
		return err
	}

	key := p.CommitProof.Keys()[0]
	err = p.CommitProof.VerifyItem(key, p.CommitValue)
	if err != nil {
		return err
	}

	commitID, err := storage.UnmarshalCommitID(p.CommitValue)
	if err != nil {
		return err
	}
	// Verify Commit in data proof
	if !reflect.DeepEqual(commitID.Hash.Bytes(), dataRoot) {
		return fmt.Errorf("CommitID hash != dataRoot, %v != %x", commitID.Hash, dataRoot)
	}

	// Verify data proof
	err = p.DataProof.Verify(dataRoot)
	if err != nil {
		return err
	}
	key = p.DataProof.Keys()[0]
	err = p.DataProof.VerifyItem(key, p.DataValues[0])
	if err != nil {
		return err
	}

	// Verify storage keys
	for i := range p.DataKeys {
		err = p.DataProof.VerifyItem(p.DataKeys[i], p.DataValues[i])
		if err != nil {
			return err
		}
	}

	return nil
}
