package dependencies

import (
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/txs"
	"github.com/hyperledger/burrow/txs/payload"
)

type SeqAccount struct {
	Account             acm.AddressableSigner
	PartitionIDSequence map[int]uint64
}

type Accounts struct {
	AccountMap    map[common.Address]*SeqAccount
	AllowedMap    map[crypto.Address]map[int64]common.Address
	TokenOwnerMap map[int64]common.Address
	lastAccountID int
}

func NewAccounts() Accounts {
	return Accounts{
		AccountMap:    make(map[common.Address]*SeqAccount),
		AllowedMap:    make(map[crypto.Address]map[int64]common.Address),
		TokenOwnerMap: make(map[int64]common.Address),
	}
}

func (ac *Accounts) GetOrCreateAccount(addr common.Address) *SeqAccount {
	if val, ok := ac.AccountMap[addr]; ok {
		return val
	}
	// logrus.Infof("Account id: %v", ac.lastAccountID)
	acc := acm.GeneratePrivateAccountFromSecret(strconv.Itoa(ac.lastAccountID))
	ac.AccountMap[addr] = &SeqAccount{
		Account:             acm.SigningAccounts([]*acm.PrivateAccount{acc})[0],
		PartitionIDSequence: make(map[int]uint64),
	}
	ac.lastAccountID++
	return ac.AccountMap[addr]

}

func (ac *Accounts) AddAllowed(from crypto.Address, to common.Address, tokenID int64) {
	if _, ok := ac.AllowedMap[from]; !ok {
		ac.AllowedMap[from] = make(map[int64]common.Address)
	}
	ac.AllowedMap[from][tokenID] = to
}

func (ac *Accounts) DeleteAllowed(addr crypto.Address, tokenID int64) {
	delete(ac.AllowedMap[addr], tokenID)
}

func (ac *Accounts) IsAllowed(addr crypto.Address, tokenID int64) bool {
	val, ok := ac.AllowedMap[addr]
	if !ok {
		return false
	}
	_, ok = val[tokenID]
	if ok {
		return true
	}
	return false
}

type TxResponse struct {
	PartitionIndex  int
	ChainID         string
	Tx              *payload.CallTx
	Signer          *SeqAccount
	MethodName      string
	OriginalIds     []int64
	OriginalBirthID int64
	AddressArgument []common.Address
	BigIntArgument  *big.Int
}

func (tr *TxResponse) Sign() *txs.Envelope {
	txPayload := payload.Payload(tr.Tx)
	tr.Signer.PartitionIDSequence[tr.PartitionIndex]++
	tr.Tx.Input.Sequence = tr.Signer.PartitionIDSequence[tr.PartitionIndex]
	env := txs.Enclose(tr.ChainID, txPayload)
	env.Sign(tr.Signer.Account)
	return env
}
