package server

import (
	"log"
)

type MsgGetData struct {
	// Address of the local peer.
	AddrSender *NetAddress
	// Address of the local peer.
	AddrReceiver *NetAddress

	ID 			[]byte
	Kind 		string
}

func (s *Server) NewMsgGetData(addrTo *NetAddress, ID []byte, kind string) *MsgGetData {
	return &MsgGetData{s.ipStatus, addrTo, ID, kind}
}

func (s *Server) sendGetData(addrTo *NetAddress, ID []byte, kind string) ([]byte, error) {
	s.Log(true, "GetData kind:"+kind+ " sent to:", addrTo.String())
	//assigne en []byte la structure getblocks
	payload := gobEncode(*s.NewMsgGetData(addrTo, ID, kind))
	//on append la commande et le payload
	request := append(commandToBytes("getdata"), payload...)
	return request, s.sendData(addrTo.String(), request)
}

func (s *Server) handleGetData(request []byte) {
	var payload MsgGetData
	if err := getPayload(request, &payload); err != nil {
		log.Panic(err)
	}
	addr := payload.AddrSender.String()
	s.Log(true, "GetData kind:"+payload.Kind+ " received from :", addr)

	if payload.Kind == "block" {
		//block
		block, _ := s.chain.GetBlockByHash(payload.ID)
		s.sendBlock(payload.AddrSender, block)
	} else {
		//tx
	}
}