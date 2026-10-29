package roefactura

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"

	efactura "github.com/invopop/gobl.ro.anaf/addon"
)

// Status tells the caller what Romania does with a document of a given kind.
type Status string

// The states a Kind can be in.
const (
	StatusAccepted    Status = "accepted"
	StatusSkipped     Status = "skipped"
	StatusUnsupported Status = "unsupported"
)

// Kind is a document type as ANAF sees it, decided before the document is
// normalized so the caller can drop what will never be filed.
type Kind struct {
	Name   string
	Code   cbc.Code
	Status Status
}

// Documents ANAF accepts, each with the UNTDID 1001 code the addon files it
// under.
var (
	KindInvoice    = Kind{Name: "invoice", Code: efactura.DocumentTypeInvoice, Status: StatusAccepted}
	KindCreditNote = Kind{Name: "credit-note", Code: efactura.DocumentTypeCreditNote, Status: StatusAccepted}
	KindCorrective = Kind{Name: "corrective", Code: efactura.DocumentTypeCorrective, Status: StatusAccepted}
	KindSelfBilled = Kind{Name: "self-billed", Code: efactura.DocumentTypeSelfBilled, Status: StatusAccepted}
	KindAccounting = Kind{Name: "accounting", Code: efactura.DocumentTypeAccounting, Status: StatusAccepted}
)

// Documents ANAF never receives.
var (
	KindProforma  = Kind{Name: "proforma", Status: StatusSkipped}
	KindDebitNote = Kind{Name: "debit-note", Status: StatusUnsupported}
	KindOther     = Kind{Name: "other", Status: StatusUnsupported}
)

// KindOf reports what Romania calls a GOBL invoice. It mirrors the addon's
// scenarios, which set the same codes once the document is calculated.
func KindOf(invoice *bill.Invoice) Kind {
	if invoice == nil {
		return KindOther
	}

	switch invoice.Type {
	case bill.InvoiceTypeStandard:
		switch {
		case efactura.IsSelfBilled(invoice):
			return KindSelfBilled
		case efactura.IsAccounting(invoice):
			return KindAccounting
		default:
			return KindInvoice
		}

	// A credit note is filed as 381 whether or not the buyer issued it, so
	// self-billing only changes the upload flag.
	case bill.InvoiceTypeCreditNote:
		return KindCreditNote

	case bill.InvoiceTypeCorrective:
		return KindCorrective

	case bill.InvoiceTypeProforma:
		return KindProforma

	case bill.InvoiceTypeDebitNote:
		return KindDebitNote

	default:
		return KindOther
	}
}
