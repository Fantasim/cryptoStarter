package blockchain

import (
	"log"

	conf "tway/config"
	wire "tway/wire"
	bolt "github.com/boltdb/bolt"
	"bytes"
)

//Structure utiliser pour parcourir les blocks de la chain
type BlockchainExplorer struct {
	CurrentHash []byte
	DB          *bolt.DB
}

func NewExplorer() *BlockchainExplorer {
	return &BlockchainExplorer{CurrentHash: BC.Tip, DB: BC.DB}
}

//Retourne le block suivant 
//commence par le block correspondant au tip
func (be *BlockchainExplorer) Next() *wire.Block{
	//Si le block est genese
	if bytes.Compare(be.CurrentHash, conf.GENESIS_BLOCK_PREVHASH) == 0 {
		return nil
	}

	var block *wire.Block

	err := be.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BLOCK_BUCKET))
		encodedBlock := b.Get(be.CurrentHash)
		block = wire.DeserializeBlock(encodedBlock)
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	
	be.CurrentHash = block.Header.HashPrevBlock
	return block
}