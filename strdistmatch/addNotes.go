package main

import (
	"github.com/nickwells/param.mod/v6/param"
)

const (
	noteBaseName = "strdistmatch - "

	noteNameAlgos = noteBaseName + "string distance algorithms"
)

// addNotes adds the notes for this program.
func addNotes(_ *Prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.AddNote(noteNameAlgos,
			"These are algorithms which give some notion of distance"+
				" between strings. They should all give a value of zero"+
				" for the distance from a string to itself. The intention"+
				" is that a smaller distance between two strings"+
				" indicates a greater similarity.")

		return nil
	}
}
