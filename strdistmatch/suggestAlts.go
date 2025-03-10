package main

import (
	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/strdist.mod/v2/strdist"
)

// SuggestAlternatives searches the population for the closest matches to the
// passed string and if any are found it returns a string suggesting the
// alternative values.
func SuggestAlternatives(n int, s string, pop []string) string {
	finder := strdist.DefaultFinders[strdist.CaseBlindAlgoNameCosine]

	alts := finder.FindNStrLike(n, s, pop...)
	if len(alts) == 0 {
		return ""
	}

	return `, did you mean ` + english.JoinQuoted(alts, ", ", " or ", `"`, `"`)
}
