package main

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/verbose.mod/verbose"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// makeParamSet generates the param set ready for parsing
func makeParamSet(prog *Prog) *param.PSet {
	return paramset.NewOrPanic(
		addParams(prog),
		addNotes(prog),
		verbose.AddParams,
		versionparams.AddParams,
		param.SetProgramDescription(
			"this will find matches to the given string from the given"+
				" population. You can give multiple different algorithms"+
				" to try, each with different parameters. The results"+
				" from each algorithm and parameter set are then"+
				" tabulated in the output. This allows you to compare"+
				" different algorithms and parameter sets and helps you"+
				" to decide which algorithm to use"),
	)
}
