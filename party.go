package roefactura

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"

	efactura "github.com/invopop/gobl.ro.anaf/addon"
)

// CIF returns the tax number as the ANAF API wants it: the bare code, without
// the RO fiscal attribute.
func CIF(party *org.Party) string {
	return efactura.CIF(party)
}

// Issuer returns the party ANAF files a document under: the enforcement body
// named as payee for an enforcement invoice, the buyer for a self-billed one,
// and the supplier otherwise.
func Issuer(invoice *bill.Invoice) *org.Party {
	switch {
	case invoice == nil:
		return nil

	case efactura.IsEnforcement(invoice):
		if invoice.Payment == nil {
			return nil
		}
		return invoice.Payment.Payee

	case efactura.IsSelfBilled(invoice):
		return invoice.Customer

	default:
		return invoice.Supplier
	}
}
