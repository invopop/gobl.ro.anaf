package efactura

import (
	"testing"

	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

func TestIsOutsideScope(t *testing.T) {
	combo := func(code cbc.Code) *tax.Combo {
		return &tax.Combo{Category: tax.CategoryVAT, Ext: tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTaxCategory: code})}
	}

	t.Run("category O on a line", func(t *testing.T) {
		invoice := &bill.Invoice{Lines: []*bill.Line{{Taxes: tax.Set{combo(en16931.TaxCategoryOutsideScope)}}}}
		if !IsOutsideScope(invoice) {
			t.Error("outside scope: want true, got false")
		}
	})

	t.Run("category O on a document level charge", func(t *testing.T) {
		invoice := &bill.Invoice{Charges: []*bill.Charge{{Taxes: tax.Set{combo(en16931.TaxCategoryOutsideScope)}}}}
		if !IsOutsideScope(invoice) {
			t.Error("outside scope: want true, got false")
		}
	})

	t.Run("standard rated only", func(t *testing.T) {
		invoice := &bill.Invoice{Lines: []*bill.Line{{Taxes: tax.Set{combo(en16931.TaxCategoryStandard)}}}}
		if IsOutsideScope(invoice) {
			t.Error("outside scope: want false, got true")
		}
	})

	t.Run("no invoice", func(t *testing.T) {
		if IsOutsideScope(nil) {
			t.Error("outside scope: want false, got true")
		}
	})
}
