package dependencies

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/enriquefynn/sharding-runner/burrow-client/logs-replayer/partitioning"
	"github.com/hyperledger/burrow/rpc/rpcquery"
	"github.com/hyperledger/burrow/txs/payload"
	log "github.com/sirupsen/logrus"
)

type Node struct {
	child map[int64]*Node
	tx    *TxResponse
}

type Dependencies struct {
	Length int
	idDep  map[int64]*Node
}

func (dp *Dependencies) bfs() {
	// visited := make(map[*Node]bool)
	for k := range dp.idDep {
		fmt.Printf("%v ", k)
	}
	fmt.Printf("\n")
}

func NewDependencies() *Dependencies {
	return &Dependencies{
		idDep: make(map[int64]*Node),
	}
}

// AddDependency add a dependency and return if is allowed to send tx
func (dp *Dependencies) AddDependency(tx *TxResponse) bool {
	newNode := &Node{
		tx:    tx,
		child: make(map[int64]*Node),
	}
	dp.Length++

	shouldWait := false
	for _, dependency := range tx.OriginalIds {
		if _, ok := dp.idDep[dependency]; !ok {
			// logrus.Infof("Adding dependency: %v -> %v (%v)", dependency, newNode.tx.methodName, newNode.tx.originalIds)
			dp.idDep[dependency] = newNode
		} else {
			shouldWait = true
			father := dp.idDep[dependency]
			// "recursive" insert dependency
			for {
				if father.child[dependency] == nil {
					// logrus.Infof("Adding dependency: %v (%v) -> %v (%v)", father, father.tx.methodName, newNode.tx.methodName, newNode.tx.originalIds)
					father.child[dependency] = newNode
					break
				}
				father = father.child[dependency]
			}
		}
	}
	return shouldWait
}

// AddDependency add a dependency and return if is allowed to send tx
func (dp *Dependencies) AddDependencyWithMoves(tx *TxResponse, part partitioning.Partitioning) []*TxResponse {
	var partitionToGo int64
	objectsToMove := make(map[int64]bool) // Objects -> should move to partitionToGo
	if len(tx.OriginalIds) == 3 {
		// Last one is the kitty id, should not be considered (create in same partition as matron)
		partitioningObjects := tx.OriginalIds[:2]
		partitionToGo = part.WhereToMove(partitioningObjects...)
		for _, dep := range partitioningObjects {
			p, exists := part.Get(dep)
			if !exists {
				log.Fatalf("GiveBirth should alter objects that exist in the partitioning")
			}
			if p != partitionToGo {
				// Have to move
				objectsToMove[dep] = true
			}
		}
		// set partition for child
		part.Move(tx.OriginalIds[2], partitionToGo)
	} else if len(tx.OriginalIds) == 2 {
		partitionToGo = part.WhereToMove(tx.OriginalIds...)
		for _, dep := range tx.OriginalIds {
			p, exists := part.Get(dep)
			if !exists {
				log.Fatalf("Breed should alter objects that exist in the partitioning")
			}
			if p != partitionToGo {
				// Have to move
				objectsToMove[dep] = true
			}
		}
	} else if len(tx.OriginalIds) == 1 {
		var exists bool
		partitionToGo, exists = part.Get(tx.OriginalIds[0])
		if !exists {
			partitionToGo = part.Add(tx.OriginalIds[0])
		}
		// partitioningObjects = tx.OriginalIds
	} else {
		log.Fatalf("Wrong txIds length: 2 -> %v %v", tx.MethodName, tx.OriginalIds)
	}

	// Set partition to go
	tx.PartitionIndex = int(partitionToGo - 1)
	tx.ChainID = strconv.Itoa(tx.PartitionIndex + 1)

	newNode := &Node{
		tx:    tx,
		child: make(map[int64]*Node),
	}
	dp.Length++

	var txsToSend []*TxResponse
	sendTx := true
	for _, dependency := range tx.OriginalIds {
		originalPartition, _ := part.Get(dependency)
		if _, ok := dp.idDep[dependency]; !ok {
			// logrus.Infof("Adding dependency: %v -> %v (%v)", dependency, newNode.tx.methodName, newNode.tx.originalIds)
			if objectsToMove[dependency] == true {
				sendTx = false
				log.Infof("Have to move %v from partition %v to partition %v", dependency, originalPartition, partitionToGo)
				// Move object
				dp.Length += 2
				part.Move(dependency, partitionToGo)
				// Add move to partitionToGo
				moves := createMoves(originalPartition, partitionToGo, dependency, tx.Tx.Input.Amount)
				txsToSend = append(txsToSend, moves.tx)
				dp.idDep[dependency] = moves
				moves.child[dependency].child[dependency] = newNode
			} else {
				dp.idDep[dependency] = newNode
			}
		} else {
			sendTx = false
			father := dp.idDep[dependency]
			// "recursive" insert dependency
			for {
				if father.child[dependency] == nil {
					// logrus.Infof("Adding dependency: %v (%v) -> %v (%v)", father, father.tx.methodName, newNode.tx.methodName, newNode.tx.originalIds)
					if objectsToMove[dependency] == true {
						// Move object
						log.Infof("Have to move %v from partition %v to partition %v!", dependency, originalPartition, partitionToGo)
						dp.Length += 2
						part.Move(dependency, partitionToGo)
						// Should move
						moves := createMoves(originalPartition, partitionToGo, dependency, tx.Tx.Input.Amount)
						father.child[dependency] = moves
						father = moves.child[dependency]
					}
					father.child[dependency] = newNode
					break
				}
				father = father.child[dependency]
			}
		}
	}
	if sendTx {
		txsToSend = append(txsToSend, tx)
	}
	return txsToSend
}

func NewTxResponse(methodName string, originalPartition, originalID int64, amount uint64) *TxResponse {
	newTx := payload.CallTx{
		Input: &payload.TxInput{
			// Address: from,
			Amount: amount,
		},
		// Address:  &to,
		Fee:      1,
		GasLimit: 4100000000,
		// Data:     data,
	}
	return &TxResponse{
		PartitionIndex: int(originalPartition - 1),
		ChainID:        strconv.Itoa(int(originalPartition)),
		Tx:             &newTx,
		MethodName:     methodName,
		OriginalIds:    []int64{originalID},
	}
}

func createMoves(originalPartition, toPartition, originalID int64, amount uint64) *Node {
	if originalPartition == toPartition {
		log.Fatalf("Cannot move to same partition from: %v to: %v", originalPartition, toPartition)
	}
	moveToNode := &Node{
		tx:    NewTxResponse("moveTo", originalPartition, originalID, amount),
		child: make(map[int64]*Node),
	}
	moveToNode.tx.BigIntArgument = big.NewInt(toPartition)
	move2Node := &Node{
		tx:    NewTxResponse("move2", toPartition, originalID, amount),
		child: make(map[int64]*Node),
	}
	moveToNode.child[originalID] = move2Node
	return moveToNode
}

func (dp *Dependencies) canSend(cameFromID int64, blockedTx *Node) bool {
	for _, otherID := range blockedTx.tx.OriginalIds {
		if otherID != cameFromID {
			// Can send?
			if dp.idDep[otherID] != blockedTx {
				return false
			}
		}
	}
	return true
}

func (dp *Dependencies) RemoveDependency(dependencies []int64) map[*TxResponse]bool {
	dp.Length--
	returnedDep := make(map[*TxResponse]bool)

	for _, dependency := range dependencies {
		blockedTx := dp.idDep[dependency].child[dependency]
		// Delete response
		delete(dp.idDep, dependency)
		if blockedTx != nil {
			// Should wait for it
			dp.idDep[dependency] = blockedTx
			// Can execute next?
			if dp.canSend(dependency, blockedTx) {
				// logrus.Infof("RETURNING %v", blockedTx.tx)
				returnedDep[blockedTx.tx] = true
			}
		}
	}
	return returnedDep
}

func (dp *Dependencies) AddFieldsToMove2(id int64, proofToGoToTx []map[int64][]*TxResponse, partitionID int, height int64, proofs *rpcquery.AccountProofs) {
	dep := dp.idDep[id]
	if dep.tx.MethodName != "move2" {
		panic("Dependency should be move2")
	}
	dep.tx.Tx.AccountProof = &proofs.AccountProof
	dep.tx.Tx.StorageProof = &proofs.StorageProof
	// Add proof and request for signer to ProofToGoTxResponse
	proofToGoToTx[partitionID][height+2] = append(proofToGoToTx[partitionID][height+2], dep.tx)
}

func (dp *Dependencies) Print() {
	fmt.Printf("IDS: ")
	for id := range dp.idDep {
		fmt.Printf("[%p] (%v %v %v) ", dp.idDep[id], id, dp.idDep[id].tx.MethodName, dp.idDep[id].tx.OriginalIds)
	}
	fmt.Printf("\n")
}
