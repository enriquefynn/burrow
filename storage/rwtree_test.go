package storage

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	hex "github.com/tmthrgd/go-hex"

	"github.com/stretchr/testify/assert"
	dbm "github.com/tendermint/tendermint/libs/db"
)

func TestSave(t *testing.T) {
	db := dbm.NewMemDB()
	rwt := NewRWTree(db, 100)
	foo := bz("foo")
	gaa := bz("gaa")
	dam := bz("dam")
	rwt.Set(foo, gaa)
	rwt.Save()
	logrus.Infof("HASH: %x version: %x", rwt.Hash(), rwt.Version())
	assert.Equal(t, gaa, rwt.Get(foo))
	rwt.Set(foo, dam)
	_, _, err := rwt.Save()
	require.NoError(t, err)
	assert.Equal(t, dam, rwt.Get(foo))
}

func TestAAA(t *testing.T) {
	db := dbm.NewMemDB()
	rwt := NewRWTree(db, 100)
	key0, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	key1, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
	key5, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000005")

	val0, _ := hex.DecodeString("0000000000000000c918f94e305e2543b7c3f3c15c35cf41cba8a5b300000001")
	val1, _ := hex.DecodeString("00005a50b310c470c614b90f294a321086318c618c63294e5394a65ad6b739ab")
	val5, _ := hex.DecodeString("000000000000000000000000ce06685804a0917a1c8b72038e768ee1e5103c2b")

	rwt.Set(key0, val0)
	rwt.Set(key1, val1)
	rwt.Set(key5, val5)
	rwt.Save()
	startKey, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	endKey, _ := hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	keys, values, _, _ := rwt.GetRangeWithProof(startKey, endKey, 1000)
	logrus.Infof("Keys %x", keys)
	logrus.Infof("Values %x", values)
	logrus.Infof("HASH: %x version: %x", rwt.Hash(), rwt.Version())
}

func TestBBB(t *testing.T) {
	startKey, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	endKey, _ := hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	db := dbm.NewMemDB()
	rwt := NewRWTree(db, 100)
	key0, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	key1, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
	key2, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000002")
	key6, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000006")

	val0, _ := hex.DecodeString("0000000000000000c918f94e305e2543b7c3f3c15c35cf41cba8a5b300000001")
	val1, _ := hex.DecodeString("00005a50b310c470c614b90f294a321086318c618c63294e5394a65ad6b739ab")
	val2, _ := hex.DecodeString("000000000000000000000000000000000000000000000000000000005caf4897")
	val6, _ := hex.DecodeString("000000000000000000000000ce06685804a0917a1c8b72038e768ee1e5103c2b")

	rwt.Set(key0, val0)
	rwt.Set(key1, val1)
	rwt.Set(key2, val2)
	rwt.Set(key6, val6)
	rwt.Delete(key6)
	hash, version, _ := rwt.Save()
	hash, version, _ = rwt.Save()
	hash, version, _ = rwt.Save()
	hash, version, _ = rwt.Save()
	hash, version, _ = rwt.Save()
	hash, version, _ = rwt.Save()
	hash, version, _ = rwt.Save()
	hash, version, _ = rwt.Save()
	// hash, version, _ = rwt.Save()
	fmt.Printf("HASH: %x version: %v\n", hash, version)
	fmt.Printf("DUMP: %v\n", rwt.Dump())
	rwt.Set(key2, val2)
	hash, version, _ = rwt.Save()
	fmt.Printf("DUMP2: %v\n", rwt.Dump())
	fmt.Printf("HASH: %x version: %v\n", hash, version)
	keys, values, proof, _ := rwt.GetRangeWithProof(startKey, endKey, 1000)
	logrus.Infof("Keys %x", keys)
	logrus.Infof("Values %x", values)
	logrus.Infof("Proof %x", proof.ComputeRootHash())
}

func TestEmptyTree(t *testing.T) {
	db := dbm.NewMemDB()
	rwt := NewRWTree(db, 100)
	fmt.Printf("%X\n", rwt.Hash())
}

func TestRollback(t *testing.T) {
	db := dbm.NewMemDB()
	rwt := NewRWTree(db, 100)
	rwt.Set(bz("Raffle"), bz("Topper"))
	_, _, err := rwt.Save()

	foo := bz("foo")
	gaa := bz("gaa")
	dam := bz("dam")
	rwt.Set(foo, gaa)
	hash1, version1, err := rwt.Save()
	require.NoError(t, err)

	// Perform some writes on top of version1
	rwt.Set(foo, gaa)
	rwt.Set(gaa, dam)
	hash2, version2, err := rwt.Save()
	rwt.Iterate(nil, nil, true, func(key, value []byte) error {
		fmt.Println(string(key), " => ", string(value))
		return nil
	})
	require.NoError(t, err)

	// Make a new tree
	rwt = NewRWTree(db, 100)
	err = rwt.Load(version1, true)
	require.NoError(t, err)
	// If you load version1 the working hash is that which you saved after version0, i.e. hash0
	require.Equal(t, hash1, rwt.Hash())

	// Run the same writes again
	rwt.Set(foo, gaa)
	rwt.Set(gaa, dam)
	hash3, version3, err := rwt.Save()
	require.NoError(t, err)
	rwt.Iterate(nil, nil, true, func(key, value []byte) error {
		fmt.Println(string(key), " => ", string(value))
		return nil
	})

	// Expect the same hashes
	assert.Equal(t, hash2, hash3)
	assert.Equal(t, version2, version3)
}

func TestVersionDivergence(t *testing.T) {
	// This test serves as a reminder that IAVL nodes contain the version and a new node is created for every write
	rwt1 := NewRWTree(dbm.NewMemDB(), 100)
	rwt1.Set(bz("Raffle"), bz("Topper"))
	hash11, _, err := rwt1.Save()
	require.NoError(t, err)

	rwt2 := NewRWTree(dbm.NewMemDB(), 100)
	rwt2.Set(bz("Raffle"), bz("Topper"))
	hash21, _, err := rwt2.Save()
	require.NoError(t, err)

	// The following 'ought' to be idempotent but isn't since it replaces the previous node with an identical one, but
	// with an incremented version number
	rwt2.Set(bz("Raffle"), bz("Topper"))
	hash22, _, err := rwt2.Save()
	require.NoError(t, err)

	assert.Equal(t, hash11, hash21)
	assert.NotEqual(t, hash11, hash22)
}

func TestMutableTree_Iterate(t *testing.T) {
	mut := NewMutableTree(dbm.NewMemDB(), 100)
	mut.Set(bz("aa"), bz("1"))
	mut.Set(bz("aab"), bz("2"))
	mut.Set(bz("aac"), bz("3"))
	mut.Set(bz("aad"), bz("4"))
	mut.Set(bz("ab"), bz("5"))
	_, _, err := mut.SaveVersion()
	require.NoError(t, err)
	mut.IterateRange(bz("aab"), bz("aad"), true, func(key []byte, value []byte) bool {
		fmt.Printf("%q -> %q\n", key, value)
		return false
	})
	fmt.Println("foo")
	mut.IterateRange(bz("aab"), bz("aad"), false, func(key []byte, value []byte) bool {
		fmt.Printf("%q -> %q\n", key, value)
		return false
	})
	fmt.Println("foo")
	mut.IterateRange(bz("aad"), bz("aab"), true, func(key []byte, value []byte) bool {
		fmt.Printf("%q -> %q\n", key, value)
		return false
	})
}
