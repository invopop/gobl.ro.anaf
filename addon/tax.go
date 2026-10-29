// Helpers for the VAT breakdown. The VAT categories come from the EN 16931 addon and the VAT
// rates from the Romanian tax regime, so this addon defines neither.
//
// Official sources:
//
//   - EN 16931-1, BR-O-02 and BR-O-04: an invoice not subject to VAT (category O) names no VAT
//     identifier for the seller or the buyer, which the converter applies

package efactura

import (
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/tax"
)

// IsOutsideScope reports an invoice carrying VAT category O anywhere in its breakdown.
func IsOutsideScope(invoice *bill.Invoice) bool {
	if invoice == nil {
		return false
	}

	for _, set := range invoiceTaxSets(invoice) {
		for _, combo := range set {
			if combo != nil && combo.Category == tax.CategoryVAT && combo.Ext.Get(untdid.ExtKeyTaxCategory) == en16931.TaxCategoryOutsideScope {
				return true
			}
		}
	}

	return false
}

// invoiceTaxSets returns every tax set of the breakdown: the lines, and the
// document level allowances and charges.
func invoiceTaxSets(invoice *bill.Invoice) []tax.Set {
	sets := make([]tax.Set, 0, len(invoice.Lines)+len(invoice.Discounts)+len(invoice.Charges))
	for _, line := range invoice.Lines {
		if line != nil {
			sets = append(sets, line.Taxes)
		}
	}
	for _, discount := range invoice.Discounts {
		if discount != nil {
			sets = append(sets, discount.Taxes)
		}
	}
	for _, charge := range invoice.Charges {
		if charge != nil {
			sets = append(sets, charge.Taxes)
		}
	}

	return sets
}
