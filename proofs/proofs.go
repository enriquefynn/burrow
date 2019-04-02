package proofs

import (
	"fmt"
	"reflect"

	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/storage"
	"github.com/sirupsen/logrus"
	amino "github.com/tendermint/go-amino"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type Proof struct {
	CommitProof *iavl.RangeProof
	DataProof   *iavl.RangeProof
	CommitValue []byte
	DataValue   []byte
}

func NewProof(commitProof, dataProof *iavl.RangeProof, commitValue, dataValue []byte) *Proof {
	return &Proof{
		CommitProof: commitProof,
		DataProof:   dataProof,
		CommitValue: commitValue,
		DataValue:   dataValue,
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
		return fmt.Errorf("CommitID hash != dataRoot, %x != %x", commitID.Hash, dataRoot)
	}

	// Verify data proof
	err = p.DataProof.Verify(dataRoot)
	if err != nil {
		return err
	}
	key = p.DataProof.Keys()[0]
	err = p.DataProof.VerifyItem(key, p.DataValue)
	if err != nil {
		return err
	}
	return nil
}

// VerifyStorageRoot verifies the commit and if the storage root proof is correct
func (p *Proof) VerifyStorageRoot(keys, values binary.Words256) error {
	// Verify Commit proof
	err := p.Verify()
	if err != nil {
		return err
	}

	key := p.CommitProof.Keys()[0]
	err = p.CommitProof.VerifyItem(key, p.CommitValue)
	if err != nil {
		return err
	}
	commitID, err := storage.UnmarshalCommitID(p.CommitValue)

	if !reflect.DeepEqual(commitID.Hash.Bytes(), SimulateStorageTree(keys, values)) {
		logrus.Infof("Wrong storage proof commit hash: %x != reconstructed storage: %x", commitID.Hash.Bytes(), SimulateStorageTree(keys, values))
		return fmt.Errorf("Wrong storage proof commit hash: %x != reconstructed storage: %x", commitID.Hash.Bytes(), SimulateStorageTree(keys, values))
	}
	return nil
}

func SimulateStorageTree(keys, values binary.Words256) []byte {
	db := dbm.NewMemDB()
	rwt := storage.NewRWTree(db, 2048)
	for i, _ := range keys {
		rwt.Set(keys[i].Bytes(), values[i].Bytes())
	}
	rwt.Save()
	return rwt.Hash()
}
