package blockchain

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"time"
	conf "tway/config"
	twayutil "tway/twayutil"
	util "tway/util"

	"github.com/boltdb/bolt"
)

//Check la validité des transactions d'un block
func (b *Blockchain) CheckBlockTXs(block *twayutil.Block) error {
	txs := block.Transactions

	//verifie la transaction coinbase
	err := b.CheckBlockTxCoinbase(block)
	if err != nil {
		return err
	}

	//on verifie individuellement la validitié de chacun des txs du block
	for _, tx := range txs {
		err := CheckIfTxIsCorrect(&tx)
		if err != nil {
			return err
		}
	}
	return nil
}

//Verifie la validité de la transaction coinbase d'un block
func (b *Blockchain) CheckBlockTxCoinbase(block *twayutil.Block) error {
	if len(block.Transactions) == 0 {
		return errors.New("any coinbase transaction in this block")
	}
	coinbaseTx := block.Transactions[0]

	if coinbaseTx.IsCoinbase() == true {
		//on recupere la totalité des outputs de la tx coinbase
		_, total_coinbase_outputs, _ := GetAmounts(&coinbaseTx)
		//on recupère la totalité des frais de transaction cumulé du block
		_, _, fees := GetTotalAmounts(block.Transactions)
		//si la totalité des outputs de la tx coinbase  correspond a la recompense
		//definis par le systeme + les frais de transaction du block
		if (total_coinbase_outputs - fees) == conf.REWARD {
			return nil
		}
		return errors.New("reward is not correct")
	}
	return errors.New("coinbase transaction is not at index 0 of transactions list")
}

//Recupere la hauteur d'un block dans la chain
func (b *Blockchain) GetBlockHeight(blockHash []byte) int {
	be := NewExplorer()
	var i = 0
	for {
		bl := be.Next()
		if bytes.Compare(bl.GetHash(), blockHash) == 0 {
			return BC.Height - i
		}
		if bl == nil {
			return -1
		}
		i++
	}
	return i
}

//Recupere un block par son hash
func (b *Blockchain) GetBlockByHash(hash []byte) (*twayutil.Block, int) {
	var block *twayutil.Block

	db := b.DB

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BLOCK_BUCKET))
		encodedBlock := b.Get(hash)
		if len(encodedBlock) == 0 {
			return errors.New("Block doesn't exist")
		}
		block = twayutil.DeserializeBlock(encodedBlock)
		return nil
	})
	if err != nil {
		return nil, -1
	}
	return block, b.GetBlockHeight(block.GetHash())
}

func (b *Blockchain) GetBlockByHeight(height int) *twayutil.Block {
	if height <= 0 || height > b.Height {
		return nil
	}

	be := NewExplorer()
	i := b.Height
	for i > height {
		be.Next()
		i--
	}
	return be.Next()
}

//Recupere le dernier block de la chain
func (b *Blockchain) GetLastBlock() *twayutil.Block {
	block, _ := b.GetBlockByHash(b.Tip)
	return block
}

func GenesisBlock(pubKey []byte) *twayutil.Block {
	//créer une nouvelle transaction coinbase
	tx := twayutil.NewCoinbaseTx(pubKey, 0)

	//créer un nouveau block et ajoute la structure
	block := &twayutil.Block{
		Transactions: []twayutil.Transaction{tx},
		Counter:      1,
	}

	//recupère le merkle root de la liste de transaction
	//contenant uniquement la transaction coinbase
	HashMerkleRoot := util.GetMerkleRoot([][]byte{tx.Serialize()}).Data

	//Créer le header du block
	header := twayutil.BlockHeader{
		Version:        []byte{conf.VERSION},
		HashPrevBlock:  conf.GENESIS_BLOCK_PREVHASH,
		HashMerkleRoot: HashMerkleRoot,
		Time:           util.EncodeInt(int(time.Now().Unix())),
		Bits:           util.EncodeInt(1),
	}
	block.Header = header
	MineBlock(block)
	return block
}

func MineBlock(b *twayutil.Block) error {
	//Créer une target de proof of work
	pow := NewProofOfWork(b)
	//cherche le nonce correspondant à la target
	nonce, _, err := pow.Run()
	if err != nil {
		return err
	}
	//ajoute le nonce au header
	b.Header.Nonce = util.EncodeInt(nonce)
	//ajoute la taille total du block
	b.Size = util.EncodeInt(int(b.GetSize()))
	return nil
}

//Retourne les informations concernant les montants de la liste de transactions
//présent dans chaque inputs et outputs
//Cette fonction retourne :
//montant total des inputs, montant total des outputs, frais de transactions
func GetTotalAmounts(list []twayutil.Transaction) (int, int, int) {
	var total_inputs = 0
	var total_outputs = 0
	var fees = 0

	for _, tx := range list {
		if tx.IsCoinbase() == false {
			total_i, total_o, fs := GetAmounts(&tx)
			total_inputs += total_i
			total_outputs += total_o
			fees += fs
		}
	}
	return total_inputs, total_outputs, fees
}

//Recupère une liste de block dans un intervalle de hauteur donné
//heightStart == hauteur initiale de recuperation
//max == heightStart + nb de block a recupere
func (b *Blockchain) GetNBlocksNextToHeight(heightStart int, max int) map[string]*twayutil.Block {
	var list = make(map[string]*twayutil.Block)

	if max > conf.MaxBlockPerMsg {
		max = conf.MaxBlockPerMsg
	}
	if heightStart > 0 {
		heightStart -= 1
	}

	be := NewExplorer()
	for i := heightStart; i < b.Height; i++ {
		block := be.Next()
		if block == nil || len(list) == max {
			break
		}
		if (b.Height - i) <= max {
			list[strconv.Itoa(i)] = block
		}
	}
	return list
}

//Verifie le contenu d'un block
func (b *Blockchain) CheckNewBlock(new *twayutil.Block) error {

	//newBlockMerkleRoot := new.Header.HashMerkleRoot
	newBlockTime := util.DecodeInt(new.Header.Time)
	newBlockMerkle := new.Header.HashMerkleRoot

	//HACK ERROR
	//if merkle root doesn't correspond to a merkle root with block's txs
	if bytes.Compare(newBlockMerkle, twayutil.GetMerkleHash(new.Transactions)) != 0 {
		return errors.New(WRONG_MERKLE_HASH)
	}

	if b.GetNewBits() != int64(util.DecodeInt(new.Header.Bits)) {
		fmt.Println("CheckNewBlock : GetNewBits", b.GetNewBits())
		fmt.Println("CheckNewBlock : new block bits", util.DecodeInt(new.Header.Bits))
		return errors.New(WRONG_BITS)
	}

	pow := NewProofOfWork(new)
	//HACK ERROR OR COMPATIBILITY VERSION ERROR
	//if proof of work is wrong
	if pow.Validate() == false {
		return errors.New(WRONG_POW_ERROR)
	}
	lastChainBlock := b.GetLastBlock()
	lastChainBlockTime := util.DecodeInt(lastChainBlock.Header.Time)
	//HACK ERROR
	//if block's time is higher than current time or less than last block added in chain
	if lastChainBlockTime > newBlockTime || time.Now().Unix() < int64(newBlockTime) {
		return errors.New(WRONG_BLOCK_TIME_ERROR)
	}
	return b.CheckBlockTXs(new)
}
