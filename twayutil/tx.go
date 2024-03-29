package twayutil

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	conf "tway/config"
	"tway/script"
	"tway/util"
)

type TxInputs struct {
	Inputs []Input
}

type Input struct {
	PrevTransactionHash []byte //[32]
	Vout                []byte //[4]
	TxInScriptLen       []byte //[1-9]
	ScriptSig           [][]byte
}

//Retourne un nouvel input de tx
func NewTxInput(prevTransactionHash []byte, vout []byte, scriptSig [][]byte) Input {
	in := Input{
		PrevTransactionHash: prevTransactionHash,
		Vout:                vout,
		TxInScriptLen:       util.EncodeInt(util.LenDoubleSliceByte(scriptSig)),
		ScriptSig:           scriptSig,
	}
	return in
}

func (in *Input) GetSize() uint64 {
	return 0
}

//Transaction -> []byte
func (in *Input) Serialize() []byte {
	b, err := json.Marshal(&in)
	if err != nil {
		log.Panic(err)
	}
	bu := new(bytes.Buffer)
	enc := gob.NewEncoder(bu)
	err = enc.Encode(b)
	if err != nil {
		log.Panic(err)
	}
	return bu.Bytes()
}

func DeserializeInput(data []byte) *Input {
	var in *Input
	var dataByte []byte

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&dataByte)
	if err != nil {
		log.Panic(err)
	}
	json.Unmarshal(dataByte, &in)
	return in
}

func (in *Input) ToInputUtil() *util.Input {
	return &util.Input{
		PrevTransactionHash: in.PrevTransactionHash,
		ScriptSig:           in.ScriptSig,
		TxInScriptLen:       in.TxInScriptLen,
		Vout:                in.Vout,
	}
}

type Output struct {
	Value          []byte //[1-8]
	TxScriptLength []byte //[1-9]
	ScriptPubKey   [][]byte
}

type TxOutputs struct {
	Outputs []Output
}

//Retourne un nouvel output de tx
func NewTxOutput(scriptPubKey [][]byte, value int) Output {
	txo := Output{
		Value:          util.EncodeInt(value),
		TxScriptLength: util.EncodeInt(util.LenDoubleSliceByte(scriptPubKey)),
		ScriptPubKey:   scriptPubKey,
	}
	return txo
}

//Si l'output a été locké avec pubKeyHash
func (output *Output) IsLockedWithPubKOrPubKH(pubKOrPubKH []byte) bool {
	scriptPubKey := output.ScriptPubKey

	//pour chaque element du script
	for _, op := range scriptPubKey {
		if bytes.Compare(op, pubKOrPubKH) == 0 {
			return true
		}
	}
	return false
}

func (out *Output) ToOutputUtil() *util.Output {
	return &util.Output{
		ScriptPubKey:   out.ScriptPubKey,
		TxScriptLength: out.TxScriptLength,
		Value:          out.Value,
	}
}

//TxOutputs -> []byte
func (outs *TxOutputs) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

//[]byte -> TxOutputs
func DeserializeTxOutputs(d []byte) *TxOutputs {
	var outs TxOutputs

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&outs)
	if err != nil {
		log.Panic(err)
	}
	return &outs
}

func (out *Output) GetSize() uint64 {
	return 0
}

type Transaction struct {
	Version    []byte //[4]
	InCounter  []byte //[1-9]
	Inputs     []Input
	OutCounter []byte //[1-9]
	Outputs    []Output
	LockTime   []byte //[4]
}

//Transaction -> []byte
func (tx *Transaction) Serialize() []byte {
	b, err := json.Marshal(tx)
	if err != nil {
		log.Panic(err)
	}
	bu := new(bytes.Buffer)
	enc := gob.NewEncoder(bu)
	err = enc.Encode(b)
	if err != nil {
		log.Panic(err)
	}
	return bu.Bytes()
}

func DeserializeTransaction(data []byte) *Transaction {
	var tx *Transaction
	var dataByte []byte

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&dataByte)
	if err != nil {
		log.Panic(err)
	}
	json.Unmarshal(dataByte, tx)
	return tx
}

//Créer une transaction coinbase
func NewCoinbaseTx(toPubKey []byte, fees int) Transaction {
	var empty [][]byte
	txIn := NewTxInput([]byte{}, util.EncodeInt(-1), empty)
	txOut := NewTxOutput(script.Script.LockingScript([][]byte{util.Ripemd160(util.Sha256(toPubKey))}, 0), conf.REWARD+fees)

	tx := Transaction{
		Version:    []byte{conf.VERSION},
		InCounter:  util.EncodeInt(1),
		OutCounter: util.EncodeInt(1),
		LockTime:   []byte{0},
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

func (tx *Transaction) GetSize() uint64 {
	return 0
}

//Signe une transaction avec le clé privé
func (tx *Transaction) Sign(prevTxs map[string]*util.Transaction, inputsPrivKey []ecdsa.PrivateKey, inputsPubKey [][]byte) {
	//si la transaction est coinbase
	if tx.IsCoinbase() {
		return
	}
	for idx, in := range tx.Inputs {
		//on signe les données
		prevTxid := hex.EncodeToString(in.PrevTransactionHash)
		r, s, err := ecdsa.Sign(rand.Reader, &inputsPrivKey[idx], prevTxs[prevTxid].Serialize())
		if err != nil {
			fmt.Println(err)
			log.Panic(err)
		}

		signature := append(r.Bytes(), s.Bytes()...)
		//on update l'input avec un nouvel input identique
		//mais comprenant le bon scriptSig
		tx.Inputs[idx] = NewTxInput(in.PrevTransactionHash, in.Vout, script.Script.UnlockingScript(signature, inputsPubKey[idx]))
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

func (tx *Transaction) GetFees(prevTxs map[string]*Transaction) int {
	if tx.IsCoinbase() == true {
		return 0
	}
	var total_input = 0
	var total_output = 0

	for _, out := range tx.Outputs {
		total_output += util.DecodeInt(out.Value)
	}

	for _, in := range tx.Inputs {
		prev := prevTxs[hex.EncodeToString(in.PrevTransactionHash)]
		for _, out := range prev.Outputs {
			total_input += util.DecodeInt(out.Value)
		}
	}
	return total_input - total_output
}

func InputsToInputsUtil(inputs []Input) []util.Input {
	var ret []util.Input
	for _, in := range inputs {
		ret = append(ret, *in.ToInputUtil())
	}
	return ret
}

func OutputsToOutputsUtil(outputs []Output) []util.Output {
	var ret []util.Output
	for _, out := range outputs {
		ret = append(ret, *out.ToOutputUtil())
	}
	return ret
}

func (tx *Transaction) ToTxUtil() *util.Transaction {
	return &util.Transaction{
		InCounter:  tx.InCounter,
		Inputs:     InputsToInputsUtil(tx.Inputs),
		OutCounter: tx.OutCounter,
		Outputs:    OutputsToOutputsUtil(tx.Outputs),
		Version:    tx.Version,
		LockTime:   tx.LockTime,
	}
}
