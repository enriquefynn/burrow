package evm

import (
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/storage"
	dbm "github.com/tendermint/tendermint/libs/db"
)

func SimulateStorageTree(keys, values binary.Words256) []byte {
	db := dbm.NewMemDB()
	rwt := storage.NewRWTree(db, 2048)
	for i, _ := range keys {
		rwt.Set(keys[i].Bytes(), values[i].Bytes())
	}
	rwt.Save()
	return rwt.Hash()
}
