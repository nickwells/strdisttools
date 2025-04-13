package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nickwells/col.mod/v4/col"
	"github.com/nickwells/col.mod/v4/colfmt"
	"github.com/nickwells/strdist.mod/v2/strdist"
	"github.com/nickwells/verbose.mod/verbose"
)

// Prog holds program parameters and status
type Prog struct {
	exitStatus int
	stack      *verbose.Stack

	maxResults int

	wordFile string

	algoParams []NamedValue[string, algoParams]
}

// NewProg returns a new Prog instance with the default values set
//
//nolint:mnd
func NewProg() *Prog {
	return &Prog{
		stack: &verbose.Stack{},

		maxResults: 5,
	}
}

// SetExitStatus sets the exit status to the new value. It will not do this
// if the exit status has already been set to a non-zero value.
func (prog *Prog) SetExitStatus(es int) {
	if prog.exitStatus == 0 {
		prog.exitStatus = es
	}
}

// ForceExitStatus sets the exit status to the new value. It will do this
// regardless of the existing exit status value.
func (prog *Prog) ForceExitStatus(es int) {
	prog.exitStatus = es
}

// Run is the starting point for the program, it should be called from main()
// after the command-line parameters have been parsed. Use the setExitStatus
// method to record the exit status and then main can exit with that status.
func (prog *Prog) Run(searchWords []string) {
	if len(searchWords) == 0 {
		fmt.Println("There are no words to search for")
		return
	}

	pop := prog.getWords()
	if len(pop) == 0 {
		fmt.Println("The population of words to be searched is empty")
		return
	}

	finders := prog.makeFinders()

	rpt := prog.makeReport(finders, searchWords)
	if rpt == nil {
		return
	}

	for _, s := range searchWords {
		vals := make([]any, 0, len(finders)*2+1)
		vals = append(vals, s)

		for _, f := range finders {
			vals = append(vals,
				f.Algo.Name(), f.Algo.Desc(),
				f.Threshold,
				f.MinStrLength,
				f.MapToLowerCase,
				f.StripRunes,
			)
			sd := f.FindLike(s, pop...)
			vals = append(vals, len(sd))

			for i := range prog.maxResults {
				if i < len(sd) {
					sdVal := sd[i]
					vals = append(vals, sdVal.Dist, sdVal.Str)
				} else {
					vals = append(vals, nil, nil)
				}
			}

			err := rpt.PrintRow(vals...)
			if err != nil {
				fmt.Printf("Cannot print the report: %s\n", err)
				prog.SetExitStatus(1)

				return
			}

			vals = []any{col.Skip{}}
		}
	}
}

// getMaxStrLen returns the maximum length of the strings in the slice
func getMaxStrLen(ss []string) uint {
	maxLen := 0

	for _, s := range ss {
		maxLen = max(maxLen, len(s))
	}

	return uint(maxLen) //nolint:gosec
}

// getMaxAlgoNameLen returns the maximum length of the Algorithm names
func getMaxAlgoNameLen(finders []*strdist.Finder) uint {
	maxLen := 0

	for _, f := range finders {
		maxLen = max(len(f.Algo.Name()), maxLen)
	}

	return uint(maxLen) //nolint:gosec
}

// getMaxAlgoDescLen returns the maximum length of the Algorithm descriptions
func getMaxAlgoDescLen(finders []*strdist.Finder) uint {
	maxLen := 0

	for _, f := range finders {
		s := f.Algo.Desc()
		sParts := strings.SplitSeq(s, "\n")

		for sp := range sParts {
			maxLen = max(maxLen, len(sp))
		}
	}

	return uint(maxLen) //nolint:gosec
}

// getMaxStripRunesLen returns the maximum length of the StripRunes value
func getMaxStripRunesLen(finders []*strdist.Finder) uint {
	maxLen := 0

	for _, f := range finders {
		maxLen = max(len(f.StripRunes), maxLen)
	}

	return uint(maxLen) //nolint:gosec
}

// makeReport generates the report for printing the results of the search
//
//nolint:mnd
func (prog *Prog) makeReport(
	finders []*strdist.Finder,
	targets []string,
) *col.Report {
	maxTargetLen := getMaxStrLen(targets)
	maxAlgoNameLen := getMaxAlgoNameLen(finders)
	maxAlgoDetailsLen := getMaxAlgoDescLen(finders)
	maxStripRunesLen := getMaxStripRunesLen(finders)

	if maxAlgoNameLen == 0 {
		maxAlgoNameLen = 1
	}

	if maxAlgoDetailsLen == 0 {
		maxAlgoDetailsLen = 1
	}

	if maxStripRunesLen == 0 {
		maxStripRunesLen = 1
	}

	h, err := col.NewHeader()
	if err != nil {
		fmt.Printf("Couldn't make the report header: %s\n", err)
		prog.SetExitStatus(1)

		return nil
	}

	targetCol := col.New(colfmt.String{W: maxTargetLen}, "target")
	cols := []*col.Col{
		col.New(
			colfmt.String{
				W: maxAlgoNameLen,
			},
			"algorithm", "name"),
		col.New(
			colfmt.WrappedString{
				W: maxAlgoDetailsLen,
			},
			"algorithm", "details"),
		col.New(
			&colfmt.Float{
				W:         9,
				Prec:      5,
				IgnoreNil: true,
			},
			"Finder", "", "threshold"),
		col.New(
			&colfmt.Int{
				W:         7,
				IgnoreNil: true,
			}, "Finder", "minimum", "str len"),
		col.New(colfmt.Bool{}, "Finder", "map to", "lower"),
		col.New(
			colfmt.String{
				W:         maxStripRunesLen,
				IgnoreNil: true,
			},
			"Finder", "strip", "runes"),
		col.New(colfmt.Int{W: 3, HandleZeroes: true}, "# of", "results"),
	}

	for i := range prog.maxResults {
		commonHeader := fmt.Sprintf("result %d", i+1)
		cols = append(cols, col.New(
			&colfmt.Float{
				W:         8,
				Prec:      4,
				IgnoreNil: true,
			},
			commonHeader, "distance"))
		cols = append(cols, col.New(
			colfmt.String{
				W:         maxTargetLen * 2,
				IgnoreNil: true,
			},
			commonHeader, "value"))
	}

	r, err := col.NewReport(h, os.Stdout, targetCol, cols...)
	if err != nil {
		fmt.Println("Couldn't create the report:", err)
		prog.SetExitStatus(1)

		return nil
	}

	return r
}

// getWords returns a slice containing the population of words to be
// searched. It will exit on any error.
func (prog *Prog) getWords() []string {
	r, err := os.Open(prog.wordFile)
	if err != nil {
		fmt.Println("Failed to open the file of words to search:", err)
		prog.SetExitStatus(1)

		return nil
	}
	defer r.Close()

	pop := []string{}

	s := bufio.NewScanner(r)
	for s.Scan() {
		pop = append(pop, s.Text())
	}

	if err := s.Err(); err != nil {
		fmt.Println("Reading the file of words to search:", err)
		prog.SetExitStatus(1)

		return nil
	}

	if len(pop) == 0 {
		fmt.Println("The file of words to search", prog.wordFile, "is empty")
		prog.SetExitStatus(1)

		return nil
	}

	verbose.Printf("the population file (%q) holds %d entries\n",
		prog.wordFile, len(pop))

	return pop
}

// makeFinders constructs the Finders from the passed parameters. It will exit
// on any error.
func (prog *Prog) makeFinders() []*strdist.Finder {
	finders := []*strdist.Finder{}

	for _, nv := range prog.algoParams {
		algoName, algoParams := nv.Name, nv.Value
		algoMaker, ok := algoMakers[algoName]

		if !ok {
			fmt.Printf("Unknown algorithm: %q\n", algoName)
			prog.SetExitStatus(1)

			return nil
		}

		algo, err := algoMaker(algoParams)
		if err != nil {
			fmt.Printf("Couldn't make the algo for %q: %s\n", algoName, err)
			prog.SetExitStatus(1)

			return nil
		}

		threshold := strdist.DefaultThresholds[algoName]
		if algoParams.useGivenThreshold {
			threshold = algoParams.threshold
		}

		fc := strdist.FinderConfig{
			Threshold:      threshold,
			MinStrLength:   algoParams.minStrLen,
			MapToLowerCase: algoParams.mapToLowerCase,
			StripRunes:     algoParams.stripRunes,
		}

		f, err := strdist.NewFinder(fc, algo)
		if err != nil {
			fmt.Printf("Couldn't make the finder for %q: %s\n", algoName, err)
			prog.SetExitStatus(1)

			return nil
		}

		finders = append(finders, f)
	}

	return finders
}
