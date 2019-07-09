package execution

import (
	"fmt"
	"testing"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/execution/evm/abi"
	"github.com/hyperledger/burrow/execution/state"
	"github.com/hyperledger/burrow/txs"
	"github.com/hyperledger/burrow/txs/payload"
	"github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tmthrgd/go-hex"
)

var (
	code1Var   = hex.MustDecodeString("6080604052600160005560898060166000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80636e9410b614602d575b600080fd5b605660048036036020811015604157600080fd5b81019080803590602001909291905050506058565b005b80f65056fea165627a7a72305820aac7f9e52448759253d7313377e5d287382200b254bbe123831d4669b1b787460029")
	code2Var   = hex.MustDecodeString("608060405260016000556002600155348015601957600080fd5b506089806100286000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80636e9410b614602d575b600080fd5b605660048036036020811015604157600080fd5b81019080803590602001909291905050506058565b005b80f65056fea165627a7a7230582020566f2966f8ee6846b87efcba920c168faf49c2abcf0a2be8be6402b14e07ca0029")
	code4Var   = hex.MustDecodeString("60806040526001600055600260015560036002556004600355348015602357600080fd5b506089806100326000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80636e9410b614602d575b600080fd5b605660048036036020811015604157600080fd5b81019080803590602001909291905050506058565b005b80f65056fea165627a7a72305820260a7e042c7a7caf22547751af98af2cb0db3591d26ae046f5107bd968686da70029")
	code8Var   = hex.MustDecodeString("608060405260016000556002600155600360025560046003556005600455600660055560076006556008600755348015603757600080fd5b506089806100466000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80636e9410b614602d575b600080fd5b605660048036036020811015604157600080fd5b81019080803590602001909291905050506058565b005b80f65056fea165627a7a72305820593a95e0b3e2e72dd84d7537e3f809742de94a0177780ab8715881e59e2e94450029")
	code16Var  = hex.MustDecodeString("6080604052600160005560026001556003600255600460035560056004556006600555600760065560086007556009600855600a600955600b600a55600c600b55600d600c55600e600d55600f600e556010600f5534801561006057600080fd5b5060898061006f6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80636e9410b614602d575b600080fd5b605660048036036020811015604157600080fd5b81019080803590602001909291905050506058565b005b80f65056fea165627a7a723058208dd429db513291e6d1b274e173ccc0a1717acc21a488442708689534939731060029")
	code32Var  = hex.MustDecodeString("6080604052600160005560026001556003600255600460035560056004556006600555600760065560086007556009600855600a600955600b600a55600c600b55600d600c55600e600d55600f600e556010600f55601160105560126011556013601255601460135560156014556016601555601760165560186017556019601855601a601955601b601a55601c601b55601d601c55601e601d55601f601e556020601f553480156100b057600080fd5b506089806100bf6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80636e9410b614602d575b600080fd5b605660048036036020811015604157600080fd5b81019080803590602001909291905050506058565b005b80f65056fea165627a7a72305820a58abd281e3bff7c309442567f2866d870e78cd18d8230649661b88a05228f2b0029")
	code64Var  = hex.MustDecodeString("6080604052600160005560026001556003600255600460035560056004556006600555600760065560086007556009600855600a600955600b600a55600c600b55600d600c55600e600d55600f600e556010600f55601160105560126011556013601255601460135560156014556016601555601760165560186017556019601855601a601955601b601a55601c601b55601d601c55601e601d55601f601e556020601f55602160205560226021556023602255602460235560256024556026602555602760265560286027556029602855602a602955602b602a55602c602b55602d602c55602e602d55602f602e556030602f55603160305560326031556033603255603460335560356034556036603555603760365560386037556039603855603a603955603b603a55603c603b55603d603c55603e603d55603f603e556040603f5534801561015057600080fd5b5060898061015f6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80636e9410b614602d575b600080fd5b605660048036036020811015604157600080fd5b81019080803590602001909291905050506058565b005b80f65056fea165627a7a723058201e51203407c0363955b85f163b883651e2e4c006d1ea24362bdacba822af00d20029")
	code128Var = hex.MustDecodeString("6080604052600160005560026001556003600255600460035560056004556006600555600760065560086007556009600855600a600955600b600a55600c600b55600d600c55600e600d55600f600e556010600f55601160105560126011556013601255601460135560156014556016601555601760165560186017556019601855601a601955601b601a55601c601b55601d601c55601e601d55601f601e556020601f55602160205560226021556023602255602460235560256024556026602555602760265560286027556029602855602a602955602b602a55602c602b55602d602c55602e602d55602f602e556030602f55603160305560326031556033603255603460335560356034556036603555603760365560386037556039603855603a603955603b603a55603c603b55603d603c55603e603d55603f603e556040603f55604160405560426041556043604255604460435560456044556046604555604760465560486047556049604855604a604955604b604a55604c604b55604d604c55604e604d55604f604e556050604f55605160505560526051556053605255605460535560556054556056605555605760565560586057556059605855605a605955605b605a55605c605b55605d605c55605e605d55605f605e556060605f55606160605560626061556063606255606460635560656064556066606555606760665560686067556069606855606a606955606b606a55606c606b55606d606c55606e606d55606f606e556070606f55607160705560726071556073607255607460735560756074556076607555607760765560786077556079607855607a607955607b607a55607c607b55607d607c55607e607d55607f607e556080607f5534801561029057600080fd5b5060898061029f6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80636e9410b614602d575b600080fd5b605660048036036020811015604157600080fd5b81019080803590602001909291905050506058565b005b80f65056fea165627a7a72305820c7cf0e5a0a198df1de2eeda816c825057aa43275a7d283e18bba1de352181d5b0029")
	code256Var = hex.MustDecodeString("6080604052600160005560026001556003600255600460035560056004556006600555600760065560086007556009600855600a600955600b600a55600c600b55600d600c55600e600d55600f600e556010600f55601160105560126011556013601255601460135560156014556016601555601760165560186017556019601855601a601955601b601a55601c601b55601d601c55601e601d55601f601e556020601f55602160205560226021556023602255602460235560256024556026602555602760265560286027556029602855602a602955602b602a55602c602b55602d602c55602e602d55602f602e556030602f55603160305560326031556033603255603460335560356034556036603555603760365560386037556039603855603a603955603b603a55603c603b55603d603c55603e603d55603f603e556040603f55604160405560426041556043604255604460435560456044556046604555604760465560486047556049604855604a604955604b604a55604c604b55604d604c55604e604d55604f604e556050604f55605160505560526051556053605255605460535560556054556056605555605760565560586057556059605855605a605955605b605a55605c605b55605d605c55605e605d55605f605e556060605f55606160605560626061556063606255606460635560656064556066606555606760665560686067556069606855606a606955606b606a55606c606b55606d606c55606e606d55606f606e556070606f55607160705560726071556073607255607460735560756074556076607555607760765560786077556079607855607a607955607b607a55607c607b55607d607c55607e607d55607f607e556080607f55608160805560826081556083608255608460835560856084556086608555608760865560886087556089608855608a608955608b608a55608c608b55608d608c55608e608d55608f608e556090608f55609160905560926091556093609255609460935560956094556096609555609760965560986097556099609855609a609955609b609a55609c609b55609d609c55609e609d55609f609e5560a0609f5560a160a05560a260a15560a360a25560a460a35560a560a45560a660a55560a760a65560a860a75560a960a85560aa60a95560ab60aa5560ac60ab5560ad60ac5560ae60ad5560af60ae5560b060af5560b160b05560b260b15560b360b25560b460b35560b560b45560b660b55560b760b65560b860b75560b960b85560ba60b95560bb60ba5560bc60bb5560bd60bc5560be60bd5560bf60be5560c060bf5560c160c05560c260c15560c360c25560c460c35560c560c45560c660c55560c760c65560c860c75560c960c85560ca60c95560cb60ca5560cc60cb5560cd60cc5560ce60cd5560cf60ce5560d060cf5560d160d05560d260d15560d360d25560d460d35560d560d45560d660d55560d760d65560d860d75560d960d85560da60d95560db60da5560dc60db5560dd60dc5560de60dd5560df60de5560e060df5560e160e05560e260e15560e360e25560e460e35560e560e45560e660e55560e760e65560e860e75560e960e85560ea60e95560eb60ea5560ec60eb5560ed60ec5560ee60ed5560ef60ee5560f060ef5560f160f05560f260f15560f360f25560f460f35560f560f45560f660f55560f760f65560f860f75560f960f85560fa60f95560fb60fa5560fc60fb5560fd60fc5560fe60fd5560ff60fe5561010060ff5534801561051157600080fd5b506089806105206000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80636e9410b614602d575b600080fd5b605660048036036020811015604157600080fd5b81019080803590602001909291905050506058565b005b80f65056fea165627a7a723058203149c238a3769b0e2cb60d3d46ed0c4e80d699614f44d78191654d8b7e4edb7b0029")
	code512Var = hex.MustDecodeString("6080604052600160005560026001556003600255600460035560056004556006600555600760065560086007556009600855600a600955600b600a55600c600b55600d600c55600e600d55600f600e556010600f55601160105560126011556013601255601460135560156014556016601555601760165560186017556019601855601a601955601b601a55601c601b55601d601c55601e601d55601f601e556020601f55602160205560226021556023602255602460235560256024556026602555602760265560286027556029602855602a602955602b602a55602c602b55602d602c55602e602d55602f602e556030602f55603160305560326031556033603255603460335560356034556036603555603760365560386037556039603855603a603955603b603a55603c603b55603d603c55603e603d55603f603e556040603f55604160405560426041556043604255604460435560456044556046604555604760465560486047556049604855604a604955604b604a55604c604b55604d604c55604e604d55604f604e556050604f55605160505560526051556053605255605460535560556054556056605555605760565560586057556059605855605a605955605b605a55605c605b55605d605c55605e605d55605f605e556060605f55606160605560626061556063606255606460635560656064556066606555606760665560686067556069606855606a606955606b606a55606c606b55606d606c55606e606d55606f606e556070606f55607160705560726071556073607255607460735560756074556076607555607760765560786077556079607855607a607955607b607a55607c607b55607d607c55607e607d55607f607e556080607f55608160805560826081556083608255608460835560856084556086608555608760865560886087556089608855608a608955608b608a55608c608b55608d608c55608e608d55608f608e556090608f55609160905560926091556093609255609460935560956094556096609555609760965560986097556099609855609a609955609b609a55609c609b55609d609c55609e609d55609f609e5560a0609f5560a160a05560a260a15560a360a25560a460a35560a560a45560a660a55560a760a65560a860a75560a960a85560aa60a95560ab60aa5560ac60ab5560ad60ac5560ae60ad5560af60ae5560b060af5560b160b05560b260b15560b360b25560b460b35560b560b45560b660b55560b760b65560b860b75560b960b85560ba60b95560bb60ba5560bc60bb5560bd60bc5560be60bd5560bf60be5560c060bf5560c160c05560c260c15560c360c25560c460c35560c560c45560c660c55560c760c65560c860c75560c960c85560ca60c95560cb60ca5560cc60cb5560cd60cc5560ce60cd5560cf60ce5560d060cf5560d160d05560d260d15560d360d25560d460d35560d560d45560d660d55560d760d65560d860d75560d960d85560da60d95560db60da5560dc60db5560dd60dc5560de60dd5560df60de5560e060df5560e160e05560e260e15560e360e25560e460e35560e560e45560e660e55560e760e65560e860e75560e960e85560ea60e95560eb60ea5560ec60eb5560ed60ec5560ee60ed5560ef60ee5560f060ef5560f160f05560f260f15560f360f25560f460f35560f560f45560f660f55560f760f65560f860f75560f960f85560fa60f95560fb60fa5560fc60fb5560fd60fc5560fe60fd5560ff60fe5561010060ff5561010161010055610102610101556101036101025561010461010355610105610104556101066101055561010761010655610108610107556101096101085561010a6101095561010b61010a5561010c61010b5561010d61010c5561010e61010d5561010f61010e5561011061010f5561011161011055610112610111556101136101125561011461011355610115610114556101166101155561011761011655610118610117556101196101185561011a6101195561011b61011a5561011c61011b5561011d61011c5561011e61011d5561011f61011e5561012061011f5561012161012055610122610121556101236101225561012461012355610125610124556101266101255561012761012655610128610127556101296101285561012a6101295561012b61012a5561012c61012b5561012d61012c5561012e61012d5561012f61012e5561013061012f5561013161013055610132610131556101336101325561013461013355610135610134556101366101355561013761013655610138610137556101396101385561013a6101395561013b61013a5561013c61013b5561013d61013c5561013e61013d5561013f61013e5561014061013f5561014161014055610142610141556101436101425561014461014355610145610144556101466101455561014761014655610148610147556101496101485561014a6101495561014b61014a5561014c61014b5561014d61014c5561014e61014d5561014f61014e5561015061014f5561015161015055610152610151556101536101525561015461015355610155610154556101566101555561015761015655610158610157556101596101585561015a6101595561015b61015a5561015c61015b5561015d61015c5561015e61015d5561015f61015e5561016061015f5561016161016055610162610161556101636101625561016461016355610165610164556101666101655561016761016655610168610167556101696101685561016a6101695561016b61016a5561016c61016b5561016d61016c5561016e61016d5561016f61016e5561017061016f5561017161017055610172610171556101736101725561017461017355610175610174556101766101755561017761017655610178610177556101796101785561017a6101795561017b61017a5561017c61017b5561017d61017c5561017e61017d5561017f61017e5561018061017f5561018161018055610182610181556101836101825561018461018355610185610184556101866101855561018761018655610188610187556101896101885561018a6101895561018b61018a5561018c61018b5561018d61018c5561018e61018d5561018f61018e5561019061018f5561019161019055610192610191556101936101925561019461019355610195610194556101966101955561019761019655610198610197556101996101985561019a6101995561019b61019a5561019c61019b5561019d61019c5561019e61019d5561019f61019e556101a061019f556101a16101a0556101a26101a1556101a36101a2556101a46101a3556101a56101a4556101a66101a5556101a76101a6556101a86101a7556101a96101a8556101aa6101a9556101ab6101aa556101ac6101ab556101ad6101ac556101ae6101ad556101af6101ae556101b06101af556101b16101b0556101b26101b1556101b36101b2556101b46101b3556101b56101b4556101b66101b5556101b76101b6556101b86101b7556101b96101b8556101ba6101b9556101bb6101ba556101bc6101bb556101bd6101bc556101be6101bd556101bf6101be556101c06101bf556101c16101c0556101c26101c1556101c36101c2556101c46101c3556101c56101c4556101c66101c5556101c76101c6556101c86101c7556101c96101c8556101ca6101c9556101cb6101ca556101cc6101cb556101cd6101cc556101ce6101cd556101cf6101ce556101d06101cf556101d16101d0556101d26101d1556101d36101d2556101d46101d3556101d56101d4556101d66101d5556101d76101d6556101d86101d7556101d96101d8556101da6101d9556101db6101da556101dc6101db556101dd6101dc556101de6101dd556101df6101de556101e06101df556101e16101e0556101e26101e1556101e36101e2556101e46101e3556101e56101e4556101e66101e5556101e76101e6556101e86101e7556101e96101e8556101ea6101e9556101eb6101ea556101ec6101eb556101ed6101ec556101ee6101ed556101ef6101ee556101f06101ef556101f16101f0556101f26101f1556101f36101f2556101f46101f3556101f56101f4556101f66101f5556101f76101f6556101f86101f7556101f96101f8556101fa6101f9556101fb6101fa556101fc6101fb556101fd6101fc556101fe6101fd556101ff6101fe556102006101ff55348015610c1157600080fd5b50608980610c206000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80636e9410b614602d575b600080fd5b605660048036036020811015604157600080fd5b81019080803590602001909291905050506058565b005b80f65056fea165627a7a723058209e6cfccbadbf0c71ca70525e244891ed0fb2e61b29236221a0c09dd74bd2f9960029")
)

func TestProofSize(t *testing.T) {
	// Init
	genesis, privAccounts, _ := deterministicGenesis.
		GenesisDoc(100, false, 1e18, 100, false, 1000)
	genesis.ChainName = "1"
	stateDB := dbm.NewDB("state", dbm.GoLevelDBBackend, "/tmp/burrow_db_test")
	defer stateDB.Close()
	st, err := state.MakeGenesisState(stateDB, genesis)
	if err != nil {
		t.Errorf("ERROR: %v", err)
	}
	err = st.InitialCommit()
	if err != nil {
		t.Errorf("ERROR: %v", err)
	}
	exe := makeExecutor(st, genesis)

	// Code
	code := code512Var
	// code := hex.MustDecodeString("6080604052348015600f57600080fd5b50600a60008190555060358060256000396000f3fe6080604052600080fdfea165627a7a723058202e2495df8f2881b7114143678324496cf665db4f9a017e8de2ff9ec1321913730029")

	tx, err := payload.NewCallTx(exe.stateCache, privAccounts[1].GetPublicKey(), nil, code, 1, 1e9, 1)
	if err != nil {
		t.Errorf("ERROR: %v", err)
	}
	err = exe.signExecuteCommitInChainID(tx, "1", privAccounts[1])
	logrus.Infof("HERE %v", err)
	if err != nil {
		t.Errorf("ERROR: %v", err)
	}

	contractAddr := crypto.NewContractAddress(tx.Input.Address, txs.Enclose("1", tx).Tx.Hash())
	contractAcc := getAccount(st, contractAddr)

	logrus.Infof("Addr: %v, balance: %v", contractAddr, contractAcc)

	valAcc := getAccount(st, privAccounts[0].GetAddress())
	logrus.Infof("Addr: %v, balance: %v", privAccounts[1].GetAddress(), valAcc)

	input := hex.MustDecodeString(hex.EncodeToString(abi.GetFunctionID("m(uint256)").Bytes()))
	shardIDToMove := binary.Int64ToWord256(2)
	input = append(input, shardIDToMove.Bytes()...)
	sequence := uint64(2)

	tx = &payload.CallTx{
		Input: &payload.TxInput{
			Address:  privAccounts[0].GetAddress(),
			Amount:   0,
			Sequence: sequence,
		},
		Address:  &contractAddr,
		GasLimit: 100000000,
		Data:     input,
	}
	sequence++
	fmt.Printf("State hash: %x version %v\n", st.Hash(), st.Version())
	err = exe.signExecuteCommitInChainID(tx, "1", privAccounts[0])

	proofs, err := st.GetAccountWithProof(contractAddr)
	if err != nil {
		t.Errorf("Error %v", err)
	}
	logrus.Infof("Size: %v %v %v", proofs.AccountProof.Size(), proofs.StorageProof.Size(), proofs.AccountProof.Size()+proofs.StorageProof.Size())
	// logrus.Infof("Error: %v %v", proofs.AccountProof, err)
	// b.ResetTimer()
}

// func TestMove2(t *testing.T) {
// 	logrus.SetFormatter(&logrus.TextFormatter{
// 		ForceColors: true,
// 	})

// 	stateDB := dbm.NewMemDB()
// 	defer stateDB.Close()
// 	genDoc := newBaseGenDoc(permission.AllAccountPermissions, permission.AllAccountPermissions)
// 	st, err := state.MakeGenesisState(stateDB, &genDoc)
// 	require.NoError(t, err)
// 	err = st.InitialCommit()
// 	require.NoError(t, err)
// 	exe := makeExecutor(st, testGenesisDoc)

// 	/*
// 		pragma solidity >0.4.25;
// 		contract Example {
// 			int v;
// 			constructor() public {
// 				v = 10;
// 			}
// 			function setV(int _v) public {
// 				v = _v;
// 			}
// 			function m(uint shardID) public {
// 				assembly {
// 					move(shardID)
// 				}
// 			}
// 			function get() public view returns(int) {
// 				return v;
// 			}
// 		}
// 	*/
// 	code := hex.MustDecodeString("608060405234801561001057600080fd5b50600a600081905550610141806100286000396000f3fe608060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806335169adf1461005c5780636d4ce63c146100975780636e9410b6146100c2575b600080fd5b34801561006857600080fd5b506100956004803603602081101561007f57600080fd5b81019080803590602001909291905050506100fd565b005b3480156100a357600080fd5b506100ac610107565b6040518082815260200191505060405180910390f35b3480156100ce57600080fd5b506100fb600480360360208110156100e557600080fd5b8101908080359060200190929190505050610110565b005b8060008190555050565b60008054905090565b80f65056fea165627a7a72305820d2640236c9372d97f88aafd85f26580584437b6469a57bed0e480073554d0dc60029")

// 	tx, _ := payload.NewCallTx(exe.stateCache, users[0].GetPublicKey(), nil, code, 1, 100000, 1)
// 	err = exe.signExecuteCommit(tx, users[0])

// 	contractAddr := crypto.NewContractAddress(tx.Input.Address, txHash(tx))
// 	contractAcc := getAccount(st, contractAddr)
// 	// contractAcc := getAccount(exe.stateCache, contractAddr)
// 	require.Equal(t, uint64(1), contractAcc.ShardID)

// 	input := hex.MustDecodeString(hex.EncodeToString(abi.GetFunctionID("m(uint256)").Bytes()))
// 	shardIDToMove := binary.Int64ToWord256(2)
// 	input = append(input, shardIDToMove.Bytes()...)
// 	sequence := uint64(2)

// 	tx = &payload.CallTx{
// 		Input: &payload.TxInput{
// 			Address:  users[0].GetAddress(),
// 			Amount:   0,
// 			Sequence: sequence,
// 		},
// 		Address:  &contractAddr,
// 		GasLimit: 100000000,
// 		Data:     input,
// 	}
// 	sequence++
// 	fmt.Printf("State hash: %x version %v\n", st.Hash(), st.Version())
// 	err = exe.signExecuteCommit(tx, users[0])
// 	fmt.Printf("State hash: %x version %v\n", st.Hash(), st.Version())
// 	require.NoError(t, err)
// 	contractAcc = getAccount(exe.stateCache, contractAddr)

// 	// Shard id should be moved to 2
// 	require.Equal(t, uint64(2), contractAcc.ShardID)
// 	logrus.Info("Contract moved to shard 2")

// 	// Try to call on a moved contract
// 	input = hex.MustDecodeString(hex.EncodeToString(abi.GetFunctionID("get()").Bytes()))
// 	tx = &payload.CallTx{
// 		Input: &payload.TxInput{
// 			Address:  users[0].GetAddress(),
// 			Amount:   0,
// 			Sequence: sequence,
// 		},
// 		Address:  &contractAddr,
// 		GasLimit: 100000000,
// 		Data:     input,
// 	}
// 	sequence++
// 	err = exe.signExecuteCommit(tx, users[0])
// 	// Should not be allowed to interact with contract
// 	require.Error(t, errors.ErrorCodeWrongShardExecution, err)

// 	// Get proof for account
// 	logrus.Infof("getting proof for contract: %v\n", contractAcc.Address)

// 	accountProof, err := st.GetKeyWithProof(st.GetKeyFormat().Account.Key(), contractAcc.Address)
// 	require.NoError(t, err)

// 	decodedAccount, err := acm.Decode(accountProof.DataValue)
// 	decodedAccount.Encode()
// 	isCorrect := accountProof.Verify()
// 	require.Equal(t, isCorrect, nil)

// 	// Get proof for storage, need to verify only the commit
// 	storageProof, err := st.GetKeyWithProof(st.GetKeyFormat().Storage.Key(), contractAcc.Address)
// 	require.NoError(t, err)

// 	storageKeyValues := binary.Int64ToWord256(0).Bytes()
// 	storageKeyValues = append(storageKeyValues, binary.Int64ToWord256(10).Bytes()...)

// 	var keys binary.Words256
// 	var values binary.Words256
// 	// Assume ordered
// 	for i := 0; i < len(storageKeyValues); i += 64 {
// 		keys = append(keys, binary.RightPadWord256(storageKeyValues[i:i+32]))
// 		values = append(values, binary.RightPadWord256(storageKeyValues[i+32:i+64]))

// 	}

// 	isCorrect = storageProof.VerifyStorageRoot(keys, values)
// 	require.Equal(t, isCorrect, nil)

// 	stateDB2 := dbm.NewMemDB()
// 	defer stateDB2.Close()
// 	var genesisShard2, privAccountsShard2, _ = deterministicGenesis.
// 		GenesisDoc(3, true, 1000, 1, true, 1000, 2)
// 	st2, err := state.MakeGenesisState(stateDB2, genesisShard2)
// 	require.NoError(t, err)
// 	err = st2.InitialCommit()
// 	require.NoError(t, err)
// 	exe2 := makeExecutor(st2, genesisShard2)

// 	input = []byte{}

// 	tx = &payload.CallTx{
// 		Input: &payload.TxInput{
// 			Address:  privAccountsShard2[0].GetAddress(),
// 			Amount:   0,
// 			Sequence: 1,
// 		},
// 		Address:  nil,
// 		GasLimit: 100000000,
// 		Data:     input,
// 		// Move
// 		AccountProof: accountProof,
// 		StorageProof: storageProof,
// 		SignedHeader: nil,
// 	}

// 	err = exe2.signExecuteCommitInChainID(tx, "2", privAccountsShard2[0])
// 	require.NoError(t, err)

// 	contractAccount, err := st2.GetAccount(contractAddr)
// 	fmt.Printf("Contract account: %v\n", contractAccount)
// 	require.NoError(t, err)

// 	// require.Equal(t, isCorrect, true)
// 	// isCorrect = accountProof.VerifyItem(accountKeyFormat.Key(decodedAccount.Address.Bytes()), movedAccount) == nil
// 	// require.Equal(t, isCorrect, true)

// 	// // Test marsheling
// 	// var proof2 = new(iavl.RangeProof)
// 	// proofBytes := make([]byte, accountProof.Size())
// 	// accountProof.MarshalTo(proofBytes)
// 	// proof2.Unmarshal(proofBytes)

// 	// storageKeyValues := binary.Int64ToWord256(0).Bytes()
// 	// storageKeyValues = append(storageKeyValues, binary.Int64ToWord256(10).Bytes()...)

// 	// // a := storageHashProof.Unmarshal()

// 	// isCorrect = proof2.Verify(stateHash) == nil
// 	// require.Equal(t, isCorrect, true)

// 	// blk1, err := st.GetBlock(1)
// 	// fmt.Printf("Block: %v\n", blk1)
// }

func (te *testExecutor) signExecuteCommitInChainID(tx payload.Payload, chainID string, signers ...acm.AddressableSigner) error {
	txEnv := txs.Enclose(chainID, tx)
	err := txEnv.Sign(signers...)
	if err != nil {
		return err
	}
	txe, err := te.Execute(txEnv)
	if err != nil {
		return err
	}
	if txe.Exception != nil {
		logrus.Infof("TXE exception: %v", txe.Exception)
		return txe.Exception
	}
	_, err = te.Commit(nil)
	return err
}
