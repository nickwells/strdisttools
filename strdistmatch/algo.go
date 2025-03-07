package main

import "github.com/nickwells/strdist.mod/v2/strdist"

// algoParams holds parameters needed to create an algo Finder
type algoParams struct {
	// the following values are used to construct the NGramConfig
	nGramLen          int
	minNGramLen       int
	overflowTheSource bool

	maxNGramCacheSize int

	// the following values are used to construct the FinderConfig
	threshold         float64
	useGivenThreshold bool
	minStrLen         int
	mapToLowerCase    bool
	stripRunes        string
}

// algoMaker is the type of a function taking an algoParams and returning a
// strdist.Algo
type algoMaker func(algoParams) (strdist.Algo, error)

var algoMakers = map[string]algoMaker{
	strdist.AlgoNameLevenshtein: func(algoParams) (strdist.Algo, error) {
		return strdist.LevenshteinAlgo{}, nil
	},
	strdist.AlgoNameScaledLevenshtein: func(algoParams) (
		strdist.Algo, error,
	) {
		return strdist.ScaledLevAlgo{}, nil
	},
	strdist.AlgoNameCosine: func(ap algoParams) (strdist.Algo, error) {
		ngc := strdist.NGramConfig{
			Length:            ap.nGramLen,
			MinLength:         ap.minNGramLen,
			OverFlowTheSource: ap.overflowTheSource,
		}
		return strdist.NewCosineAlgo(ngc, ap.maxNGramCacheSize)
	},
	strdist.AlgoNameHamming: func(_ algoParams) (strdist.Algo, error) {
		return strdist.HammingAlgo{}, nil
	},
	strdist.AlgoNameJaccard: func(ap algoParams) (strdist.Algo, error) {
		ngc := strdist.NGramConfig{
			Length:            ap.nGramLen,
			MinLength:         ap.minNGramLen,
			OverFlowTheSource: ap.overflowTheSource,
		}
		return strdist.NewJaccardAlgo(ngc, ap.maxNGramCacheSize)
	},
	strdist.AlgoNameWeightedJaccard: func(ap algoParams) (
		strdist.Algo, error,
	) {
		ngc := strdist.NGramConfig{
			Length:            ap.nGramLen,
			MinLength:         ap.minNGramLen,
			OverFlowTheSource: ap.overflowTheSource,
		}
		return strdist.NewWeightedJaccardAlgo(ngc, ap.maxNGramCacheSize)
	},
}
