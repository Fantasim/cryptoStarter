package blockchain

import (
	"log"
	"crypto/ecdsa"
	"crypto/rand"
	"letsgo/script"
	"bytes"
	"encoding/gob"
	"letsgo/util"
	"letsgo/wallet"
	"fmt"
)

type Transaction struct {
	Version []byte //[4]
	InCounter []byte //[1-9]
	Inputs []Input
	OutCounter []byte //[1-9]
	Outputs []Output
	LockTime []byte //[4]
}

//Transaction -> []byte
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

//Créer une transaction coinbase
func NewCoinbaseTx(toPubKey []byte, fees int) Transaction {
	var empty [][]byte
	txIn := NewTxInput([]byte{}, util.EncodeInt(-1), empty)
	txOut := NewTxOutput(script.Script.LockingScript(wallet.HashPubKey(toPubKey)), REWARD + fees)

	tx := Transaction{
		Version: []byte{VERSION},
		InCounter: util.EncodeInt(1),
		OutCounter: util.EncodeInt(1),
		LockTime: []byte{0},
	}
	tx.Inputs = []Input{txIn}
	tx.Outputs = []Output{txOut}
	return tx
}

// Retourne l'ID de la transaction
func (tx *Transaction) GetHash() []byte {
	return util.Sha256(tx.Serialize())
}

//Retourne la valeur total des outputs de la TX
func (tx *Transaction) GetValue() int {
	val := 0
	for _, out := range tx.Outputs {
		val += util.DecodeInt(out.Value)
	}
	return val
}

//Retourne true si la tx est coinbase
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].PrevTransactionHash) == 0 && bytes.Compare(tx.Inputs[0].Vout, util.EncodeInt(-1)) == 0
}

//Récupère une transaction par son hash, avec le block dans lequel
//se trouve la transaction, ainsi que la hauteur du block
func GetTxByHash(hash []byte) (*Transaction, *Block, int) {
	be := NewExplorer()
	var i = BC_HEIGHT
	for i > 0 {
		block := be.Next()
		for _, tx := range block.Transactions {
			if bytes.Compare(hash, tx.GetHash()) == 0 {
				return &tx, block, i
			}
		}
		i--
	}
	return nil, nil, -1
}

func (tx *Transaction) GetSize() uint64 {
	return 0
}

//Créer une transaction
func CreateTx(from string, to string, amount int, fees int) *Transaction {
	var inputs []Input
	var inputsPubKey []string
	var outputs []Output
	var localUnspents []LocalUnspentOutput
	var amountGot int

	if wallet.IsAddressValid(to) == false {
		fmt.Println("Address to send is not a valid address")
		return nil
	}

	//On récupère la clé public hashée à partir de l'address 
	//à qui on envoie
	toPubKeyHash := wallet.GetPubKeyHashFromAddress([]byte(to))

	if from == "" {
		//on récupère une liste d'output qui totalise le montant a envoyer
		//on récupère aussi amountGot, qui est le total de la somme de value des outputs
		//Cette variable est indispensable, car si la valeur total obtenu est supérieur
		//au montant d'envoie, on doit transferer l'excédant sur le wallet du créateur de la tx
		amountGot, localUnspents = Walletinfo.GetLocalUnspentOutputs(amount + fees, to)
	} else {
		if wallet.IsAddressValid(from) == false {
			fmt.Println("sender address is not a valid address")
			return nil
		}
		amountGot, localUnspents = UTXO.GetLocalUnspentOutputsByPubKeyHash(wallet.GetPubKeyHashFromAddress([]byte(from)), amount + fees)
	}

	//Si le montant d'envoie est inférieur au total des wallets locaux
	if (from == "" && (amount + fees) > Walletinfo.Amount) || (from != "" && (amount + fees) > amountGot) {
		log.Println("You don't have enough coin to perform this transaction.")
		return nil
	}

	//Pour chaque output
	for _, localUs := range localUnspents {
		var emptyScript [][]byte
		//on génère un input à partir de l'output
		input := NewTxInput(localUs.TxID, util.EncodeInt(localUs.Output), emptyScript)
		//et on l'ajoute à la liste
		inputs = append(inputs, input)
		//on ajoute dans un tableau de string la clé publique correspondant 
		//au wallet proprietaire de l'output permettant la création de 
		//cette input.
		//Ce tableau de clé publique permettra de signer chaque input.	
		inputsPubKey = append(inputsPubKey, string(localUs.Wallet.PublicKey))
	}

	//on génére l'output vers l'address de notre destinaire
	out := NewTxOutput(script.Script.LockingScript(toPubKeyHash), amount)
	outputs = append(outputs, out)
	
	//Si le montant récupére par les wallets locaux est supérieur
	//au montant que l'on décide d'envoyer
	if amountGot > (amount + fees) {
		//on utilise la clé public du dernier output ajouté à la liste
		fromPubKeyHash := wallet.HashPubKey(localUnspents[len(localUnspents) - 1].Wallet.PublicKey)
		//on génére un output vers le dernier output de la liste d'utxo récupéré
		//et on envoie l'excédant
		exc := NewTxOutput(script.Script.LockingScript(fromPubKeyHash), amountGot - (amount + fees))
		outputs = append(outputs, exc)
	}

	tx := &Transaction{
		Version: []byte{VERSION},
		InCounter: util.EncodeInt(len(inputs)),
		Inputs: inputs,
		OutCounter: util.EncodeInt(len(outputs)),
		Outputs: outputs,
	}

	//on signe la transaction
	tx.Sign(localUnspents, inputsPubKey)
	return tx
	
}

//Signe une transaction avec le clé privé
func (tx *Transaction) Sign(localUnspents []LocalUnspentOutput, inputsPubKey []string){
	//si la transaction est coinbase
	if tx.IsCoinbase(){
		return
	}
	
	prevTXs := make(map[string]*Transaction)
	//on récupère la liste des transactions précédant
	//la liste des inputs de la tx
	for _, in := range tx.Inputs {
		prevTxID, prevTx := in.GetPrevTx()
		prevTXs[prevTxID] = prevTx
	}

	//on fait une copie de la transaction
	var txCopy *Transaction
	txCopy = tx
	
	for idx, _ := range txCopy.Inputs {
		//on transforme la transaction en string
		dataToSign := fmt.Sprintf("%x\n", txCopy)

		//on signe les données
		r, s, err := ecdsa.Sign(rand.Reader, &localUnspents[idx].Wallet.PrivateKey, []byte(dataToSign))
		if err != nil {
			fmt.Println(err)
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)
		//on update l'input avec un nouvel input identique 
		//mais comprenant le bon scriptSig
		pubKeyHash := wallet.HashPubKey([]byte(inputsPubKey[idx]))
		tx.Inputs[idx] = NewTxInput(tx.Inputs[idx].PrevTransactionHash, tx.Inputs[idx].Vout, script.Script.UnlockingScript(signature, pubKeyHash))
	}
}

//[]Transaction -> [][]byte
func TransactionToByteDoubleArray(txs []Transaction) [][]byte {
	ret := make([][]byte, len(txs))
	for idx, tx := range txs {
		ret[idx] = tx.Serialize()
	}
	return ret
}

//Retourne les informations concernant les montants de la transaction
//présent dans les inputs ou outputs
//Cette fonction retourne :
//montant total des inputs, montant total des outputs, frais de transactions
func (tx *Transaction) GetAmounts() (int, int, int) {

	var total_inputs = 0
	var total_outputs = 0

	if tx.IsCoinbase() == true {
		return 0,0,0
	}

	//Pour chaque input de la tx
	for _, in := range tx.Inputs {
		//on recupere la transaction précédante de l'input
		prevTx, _, _ := GetTxByHash(in.PrevTransactionHash)
		//on récupère l'output ayant permis la création de l'input
		out := prevTx.Outputs[(util.DecodeInt(in.Vout))]
		//on ajoute le montant au montant total assemblés par les inputs
		total_inputs += util.DecodeInt(out.Value)
	}
	//pour chaque output de la tx
	for _, out := range tx.Outputs {
		//on ajoute le montant au montant total redistribué vers une adresse.
		total_outputs += util.DecodeInt(out.Value)
	}
	return total_inputs, total_outputs, total_inputs - total_outputs
}
