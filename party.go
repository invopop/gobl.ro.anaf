package roefactura

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"

	efactura "github.com/invopop/gobl.ro.anaf/addon"
)

// CIF returns the tax number as the ANAF API wants it: the bare code, without
// the RO fiscal attribute. A party with no tax number, such as an enforcement
// body named by its legal registration identifier (BR-RO-130), gives that code
// instead, unless it is a personal number.
func CIF(party *org.Party) string {
	if cif := efactura.CIF(party); cif != "" {
		return cif
	}

	identity := efactura.LegalIdentity(party)
	if identity == nil || cbc.NormalizeUpperCode(identity.Type) == efactura.IdentityTypeCNP {
		return ""
	}

	return strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(identity.Code.String())), l10n.RO.String())
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
