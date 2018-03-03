package wallet

import (
	b "tway/blockchain"
	"tway/util"
)

//Structure représentant les informations liées 
//à un UTXO présent dans un wallet local
type LocalUnspentOutput struct {
	TxID []byte
	Idx int
	Amount int
	W *Wallet
}

//Récupère une liste d'outputs locaux non dépensé locké avec le pubKeyHash
//d'un montant supérieur ou égal au montant passé en paramètre
func GetLocalUnspentOutputsByPubKeyHash(pubKeyHash []byte, amount int) (int, []LocalUnspentOutput) {
	utxo := b.UTXO
	var list []LocalUnspentOutput
	w := GetWalletByPubKeyHash(pubKeyHash)

	if w == nil {
		return 0, list
	}

	amount, unspents := utxo.GetUnspentOutputsByPubKeyHash(pubKeyHash, amount)
	
	for _, us := range unspents {
		localUXO := LocalUnspentOutput{us.TxID, us.Idx, util.DecodeInt(us.Output.Value), w}
		list = append(list, localUXO)
	}

	return amount, list
}

//Récupère une liste UTXO sur des wallets 
//enregistrés localement.
func (wInfo *WalletInfo) GetLocalUnspentOutputs(amount int, notAcceptedAddr... string) (int, []LocalUnspentOutput)  {
	utxo := b.UTXO
	var total = 0
	var localUnSpents []LocalUnspentOutput

	BrowseWallet:
	for _, ws := range wInfo.Ws {

		if amount < total {
			break
		}
		for _, addr := range notAcceptedAddr {
			if addr == string(ws.Address) {
				continue BrowseWallet;
			}
		}

		a, outs := utxo.GetUnspentOutputsByPubKeyHash(HashPubKey(ws.W.PublicKey), amount - total)
		total += a
		for _, uo := range outs {
			luo := LocalUnspentOutput{uo.TxID, uo.Idx, util.DecodeInt(uo.Output.Value), ws.W}
			localUnSpents = append(localUnSpents, luo)
		} 
	}
	return total, localUnSpents
}