package main

import (
	"fmt"
	"strings"
)

func main() {
	s := "@asddf sd  as"
	s = strings.ReplaceAll(s, "  ", " ")
	s1 := strings.Split(s, " ")

	s1[0] = strings.ToUpper(s1[0])
	for _, x := range s1 {
		fmt.Println(">> ", x)
	}
}
