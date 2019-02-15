package contexts

import (
	"time"

	"github.com/hyperledger/burrow/genesis"
)

// Execution's sufficient view of blockchain
type Blockchain interface {
	BlockHash(height uint64) []byte
	LastBlockTime() time.Time
	ShardID() uint64
	Validators() []genesis.Validator
	BlockchainHeight
}

type BlockchainHeight interface {
	LastBlockHeight() uint64
}
