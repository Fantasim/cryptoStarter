package server

import (
	"io/ioutil"
	"net"
	"log"
	"fmt"
	conf "tway/config"
)

//function appelé lorsqu'une nouvelle connexion est detectée
func (s *Server) HandleConnexion(conn net.Conn) {
	//on recupere le []byte dans request
	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	command := bytesToCommand(request[:conf.CommandLength])
	switch command {
	case "addr":
		s.handleAddr(request)
	/*case "block":
		handleBlock(request)
	case "inv":
		handleInv(request, bc)
*/
	case "getaddr":
		s.handleAskAddr(request)
	case "getblocks":
		s.handleAskBlocks(request)

/*	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)*/
	case "verack":
		s.handleVerack(request)
	case "version":
		s.handleVersion(request)
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}