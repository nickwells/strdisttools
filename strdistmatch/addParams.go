package main

import (
	"fmt"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
	"github.com/nickwells/strdist.mod/v2/strdist"
)

const (
	paramNameWordFile   = "word-file"
	paramNameAlgo       = "algo"
	paramNameToLower    = "to-lower"
	paramNamePageSize   = "page-size"
	paramNameMinStrLen  = "min-str-len"
	paramNameMaxLength  = "max-length"
	paramNameMaxResults = "max-results"
)

func addParams(prog *Prog) param.PSetOptFunc {
	const (
		dfltNGramLen          = 3
		dftlMaxNGramCacheSize = 3
	)

	return func(ps *param.PSet) error {
		ps.Add(paramNameWordFile,
			psetter.Pathname{
				Value:       &prog.wordFile,
				Expectation: filecheck.FileNonEmpty(),
			},
			"the name of the file containing the population of words to"+
				" be searched",
			param.Attrs(param.MustBeSet),
		)

		algoDetailsAVals := psetter.AllowedVals[string]{
			strdist.AlgoNameScaledLevenshtein: "a scaled Levenshtein algorithm",
			strdist.AlgoNameLevenshtein:       "a Levenshtein algorithm",
			strdist.AlgoNameCosine:            "a cosine algorithm",
			strdist.AlgoNameHamming:           "a Hamming algorithm",
			strdist.AlgoNameJaccard:           "a Jaccard algorithm",
			strdist.AlgoNameWeightedJaccard:   "a weighted Jaccard algorithm",
		}
		algoDetailsSetter := ListMultiSetter[string, algoParams]{
			Value: &prog.algoParams,
			MultiSetterBase: MultiSetterBase[string, algoParams]{
				DfltEntryVal: algoParams{
					nGramLen:          dfltNGramLen,
					maxNGramCacheSize: dftlMaxNGramCacheSize,
				},
				AVals: algoDetailsAVals,
			},
		}
		algoDetailsSetter.EntryValSetterMap = map[string]EntryValSetter{
			"nGramLen": {
				Setter: psetter.Int[int]{
					Value: &algoDetailsSetter.EntryVal.nGramLen,
					Checks: []check.ValCk[int]{
						check.ValGT(0),
					},
				},
			},
			"minNGramLen": {
				Setter: psetter.Int[int]{
					Value: &algoDetailsSetter.EntryVal.minNGramLen,
					Checks: []check.ValCk[int]{
						check.ValGT(0),
					},
				},
			},
			"overflowNGrams": {
				Setter: psetter.Bool{
					Value: &algoDetailsSetter.EntryVal.overflowTheSource,
				},
			},
			"threshold": {
				Setter: psetter.Float[float64]{
					Value: &algoDetailsSetter.EntryVal.threshold,
				},
				PostActionFuncs: []MultiSetterActionFunc{
					func(_, _ string) error {
						algoDetailsSetter.EntryVal.useGivenThreshold = true
						return nil
					},
				},
			},
			"minStrLen": {
				Setter: psetter.Int[int]{
					Value: &algoDetailsSetter.EntryVal.minStrLen,
				},
			},
			"mapToLowerCase": {
				Setter: psetter.Bool{
					Value: &algoDetailsSetter.EntryVal.mapToLowerCase,
				},
			},
			"stripRunes": {
				Setter: psetter.String[string]{
					Value: &algoDetailsSetter.EntryVal.stripRunes,
					Checks: []check.ValCk[string]{
						func(s string) error {
							runeIdx := map[rune]int{}
							runeSlc := []rune(s)
							for i, r := range runeSlc {
								if idx, ok := runeIdx[r]; ok {
									return fmt.Errorf(
										"%q contains duplicate runes:"+
											" %q appears at both"+
											" the %d%s and %d%s positions",
										s, r,
										i+1, english.OrdinalSuffix(i+1),
										idx+1, english.OrdinalSuffix(idx+1))
								}
							}
							return nil
						},
					},
				},
			},
		}
		algoDetailsSetter.EntryValSMAliases = map[string]string{
			"nGramLength":    "nGramLen",
			"ngLength":       "nGramLen",
			"ngLen":          "nGramLen",
			"minNGramLength": "minNGramLen",
			"ngMinLen":       "minNGramLen",
			"ngMinLength":    "minNGramLen",
			"overflow":       "overflowNGrams",
			"Overflow":       "overflowNGrams",
			"Threshold":      "threshold",
			"toLower":        "mapToLowerCase",
			"mapToLowercase": "mapToLowerCase",
			"stripChars":     "stripRunes",
		}

		ps.Add(paramNameAlgo,
			&algoDetailsSetter,
			"the algorithm and associated details",
			param.Attrs(param.MustBeSet),
		)

		ps.Add(paramNameMaxResults,
			psetter.Int[int]{
				Value:  &prog.maxResults,
				Checks: []check.ValCk[int]{check.ValGT(0)},
			},
			"the maximum number of results to show.",
		)

		_ = ps.SetNamedRemHandler(param.NullRemHandler{}, "strings to match")

		return nil
	}
}
