package blockchain

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"log"
	"tway/script"
	"tway/twayutil"
	"tway/util"

	"github.com/boltdb/bolt"
)

const (
	UTXO_BUCKET = "chainstate"
)

var (
	UTXO *UTXOSet
)

type UTXOSet struct {
}

//Structure représentant les informations liés à un UTXO
type UnspentOutput struct {
	TxID     []byte
	Idx      int //index of output in tx
	Output   twayutil.Output
	MultiSig bool
}

type UnspentOutputs struct {
	Outputs []UnspentOutput
}

//TxOutputs -> []byte
func (outs *UnspentOutputs) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

//[]byte -> TxOutputs
func DeserializeTxOutputs(d []byte) *UnspentOutputs {
	var outs UnspentOutputs

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&outs)
	if err != nil {
		log.Panic(err)
	}
	return &outs
}

//Récupère un output non dépensé se trouvant dans txHash a la position vout
//Retourne nil si non existant.
func (utxo *UTXOSet) GetUnSpentOutputByVoutAndTxHash(vout int, txHash []byte) *UnspentOutput {
	db := BC.DB
	var unspentOutput *UnspentOutput = nil

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(UTXO_BUCKET))
		encodedTxOutputs := b.Get(txHash)
		if len(encodedTxOutputs) == 0 {
			return errors.New("UTXO doesn't exist")
		}
		unSpents := DeserializeTxOutputs(encodedTxOutputs)
		for _, out := range unSpents.Outputs {
			if out.Idx == vout {
				unspentOutput = &out
				return nil
			}
		}
		return nil
	})
	return unspentOutput
}

//Récupère une liste d'outputs non dépensé locké avec
//une clé publique hashé (Pay2PubKH) ou une clé publique (Pay2ScriptH)
//d'un montant supérieur ou égal au montant passé en paramètre
func (utxo *UTXOSet) GetUnspentOutputsByPubKOrPubKH(pubKOrPubKHList [][]byte, amount int) (int, []UnspentOutput) {
	var unspentOutputs []UnspentOutput
	accumulated := 0
	db := BC.DB

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(UTXO_BUCKET))
		c := b.Cursor()

		//Pour chaque transaction comportant des outputs non dépensés
		for k, v := c.First(); k != nil; k, v = c.Next() {
			unSpents := DeserializeTxOutputs(v)
			//pour chaque output non dépnesé de la tx
			for _, unSpent := range unSpents.Outputs {
				//si l'output est locké avec la pubKeyHash passé en paramètre
				//et que le montant accumulé est inférieur au montant passé en paramètre
				for _, pubKOrPubKH := range pubKOrPubKHList {
					islocked := unSpent.Output.IsLockedWithPubKOrPubKH(pubKOrPubKH)
					if islocked == true && accumulated < amount {
						value := util.DecodeInt(unSpent.Output.Value)
						accumulated += value
						//on ajoute l'output à la liste des utxo
						unspentOutputs = append(unspentOutputs, unSpent)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return accumulated, unspentOutputs
}

//Reindex la liste des utxo dans le bucket des UTXOS
func (utxo *UTXOSet) Reindex() error {
	bucketName := []byte(UTXO_BUCKET)
	db := BC.DB
	UTXO := BC.FindUTXO()

	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != nil && err != bolt.ErrBucketNotFound {
			return err
		}
		_, err = tx.CreateBucket(bucketName)
		if err != nil {
			return err
		}
		b := tx.Bucket(bucketName)
		for txID, outs := range UTXO {
			key, _ := hex.DecodeString(txID)
			err = b.Put(key, outs.Serialize())
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func OutputToUnspentOutput(out *twayutil.Output, tx *twayutil.Transaction, vout int) UnspentOutput {
	return UnspentOutput{
		TxID:     tx.GetHash(),
		Output:   *out,
		Idx:      vout,
		MultiSig: script.Script.IsPayToHashScript(out.ScriptPubKey),
	}
}

//Compte le nombre de transaction contenant des outputs non dépensés
func (utxo *UTXOSet) CountTx() int {
	bucketName := []byte(UTXO_BUCKET)
	db := BC.DB
	var i = 0
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			i++
		}
		return nil
	})
	return i
}
