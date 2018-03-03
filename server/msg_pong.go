package server

import (
	"log"
)

type MsgPong struct {
	// Address of the local peer.
	AddrSender *NetAddress
	// Address of the local peer.
	AddrReceiver *NetAddress
}

func (s *Server) NewPong(addrTo *NetAddress) *MsgPong {
	return &MsgPong{s.ipStatus, addrTo}
}

//envoie une requete pong (reponse à un ping)
func (s *Server) sendPong(addrTo *NetAddress) ([]byte, error) {
	addr := addrTo.String()

	s.Log(true, "Pong sent to:", addr)
	payload := gobEncode(*s.NewPong(addrTo))
	request := append(commandToBytes("pong"), payload...)
	return request, s.sendData(addrTo.String(), request)
}

//Receptionne une requete pong (reponse d'un ping)
func (s *Server) handlePong(request []byte) {
	var payload MsgPong
	if err := getPayload(request, &payload); err != nil {
		log.Panic(err)
	}
	addr := payload.AddrSender.String()
	s.Log(true, "Pong received from :", addr)
	s.peers[addr].PongReceived()
	s.peers[addr].IncreaseBytesReceived(uint64(len(request)))
}