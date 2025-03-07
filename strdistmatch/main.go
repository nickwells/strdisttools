package main

// strdistmatch

import (
	"os"
)

// Created: Thu Feb  4 15:59:19 2021

func main() {
	prog := NewProg()
	ps := makeParamSet(prog)
	ps.Parse()

	prog.Run(ps.Remainder())
	os.Exit(prog.exitStatus)
}
