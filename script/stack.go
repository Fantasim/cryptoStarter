package script

import (
	"fmt"
	"strconv"
	"errors"
	"letsgo/util"
)

type stack struct {
	stk               [][]byte
}

//Affiche la stack
func (s *stack) PrintStack(){
	fmt.Print("stack : ")
	for _, elem := range s.stk {
		fmt.Print(elem)
	}
	fmt.Print("\n")
}

//ajoute un []byte dans la stack
func (s *stack) Push(elem []byte) error {
	s.stk = append(s.stk, elem)
	return nil
}

//ajoute un int dans la stack
func (s *stack) PushInt(n int) error {
	s.stk = append(s.stk, []byte(strconv.Itoa(n)))
	return nil
}

//ajoute un bool dans la stack
func (s *stack) PushBool(b bool) error {
	if b {
		s.stk = append(s.stk, []byte{1})
	} else {
		s.stk = append(s.stk, nil)
	}
	return nil
}

//recupere et supprime le dernier element ajouté dans la stack
//format : []byte
func (s *stack) Pop() ([]byte, error) {
	if len(s.stk) == 0 {
		return []byte{}, errors.New("empty stack")
	}
	var idx = len(s.stk) - 1
	var ret = s.stk[idx]
	s.stk = append(s.stk[:idx], s.stk[idx+1:]...)
	return ret, nil
}

//recupere et supprime le dernier element ajouté dans la stack
//format : int
func (s *stack) PopInt() (int, error){
	if len(s.stk) == 0 {
		return 0, errors.New("empty stack")
	}
	var idx = len(s.stk) - 1

	ret, err := util.ArrayByteToInt(s.stk[idx])
	if err != nil {
		return 0, err
	}

	s.stk = append(s.stk[:idx], s.stk[idx+1:]...)
	return ret, nil
}

//recupere et supprime le dernier element ajouté dans la stack
//format : bool
func (s *stack) PopBool() (bool, error){
	var idx = len(s.stk) - 1
	if len(s.stk) == 0 {
		return false, errors.New("empty stack")
	}

	ret, err := util.ArrayByteToInt(s.stk[idx])
	if err != nil {
		fmt.Println(s.stk[idx])
		ret = int(s.stk[idx][0])
	}

	s.stk = append(s.stk[:idx], s.stk[idx+1:]...)
	if ret == 1 {
		return true, nil
	}
	return false, nil
}

//Duplique les n derniers elements de la stack
func (s *stack) DupN(n int)error{
	for n > 0 {
		s.stk = append(s.stk, s.stk[len(s.stk) - 1])
		n--
	}
	return nil
}