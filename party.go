package roefactura

import (
	"strings"

	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"

	efactura "github.com/invopop/gobl.ro.anaf/addon"
)

// CIF returns the tax number as the ANAF API wants it: the bare code, without
// the RO fiscal attribute.
func CIF(party *org.Party) string {
	if !efactura.HasTaxNumber(party) {
		return ""
	}

	code := strings.ToUpper(strings.TrimSpace(party.TaxID.Code.String()))
	if party.TaxID.Country.Code() == l10n.RO {
		code = strings.TrimPrefix(code, l10n.RO.String())
	}

	return code
}
