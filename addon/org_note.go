package efactura

import (
	"fmt"

	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

// Lengths BR-RO-L300 and BR-RO-L100 put on note text, at document and line level.
const (
	maxNoteText          = 300
	maxNoteAccountingRef = 100
)

// NoteKeyBuyerAccountingRef routes a line note into BT-133 rather than BT-127,
// which is why it takes the tighter limit.
const NoteKeyBuyerAccountingRef cbc.Key = "buyer-accounting-ref"

func orgNoteRules() *rules.Set {
	return rules.For(new(org.Note),
		rules.Assert("01", "note (BT-22, BT-127), with its subject qualifier when set, must be no more than 300 characters long (BR-RO-L300)",
			is.Func("note within length", noteWithinLength),
		),
		rules.Assert("02", "invoice line buyer accounting reference (BT-133) must be no more than 100 characters long (BR-RO-L100)",
			is.Func("accounting reference within length", accountingRefWithinLength),
		),
	)
}

// RenderNote writes BT-22 and BT-127 the way gobl.ubl does, prefixing the text
// with the UNTDID 4451 subject when the note carries one.
func RenderNote(note *org.Note) string {
	if note == nil {
		return ""
	}

	if code := note.Ext.Get(untdid.ExtKeyTextSubject); code != cbc.CodeEmpty {
		return fmt.Sprintf("#%s#%s", code, note.Text)
	}

	return note.Text
}

func noteWithinLength(value any) bool {
	note, ok := value.(*org.Note)
	if !ok || note == nil {
		return true
	}

	return len([]rune(RenderNote(note))) <= maxNoteText
}

// accountingRefWithinLength covers BT-133, written from the note text alone.
func accountingRefWithinLength(value any) bool {
	note, ok := value.(*org.Note)
	if !ok || note == nil || note.Key != NoteKeyBuyerAccountingRef {
		return true
	}

	return len([]rune(note.Text)) <= maxNoteAccountingRef
}
