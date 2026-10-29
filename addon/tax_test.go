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

func TestOutsideScopeStandsAlone(t *testing.T) {
	outsideScope := &tax.Combo{Category: tax.CategoryVAT, Ext: tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTaxCategory: en16931.TaxCategoryOutsideScope})}
	standard := &tax.Combo{Category: tax.CategoryVAT, Ext: tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTaxCategory: en16931.TaxCategoryStandard})}

	tests := []struct {
		name  string
		lines []tax.Set
		want  bool
	}{
		{name: "only standard rated lines", lines: []tax.Set{{standard}}, want: true},
		{name: "only outside scope lines", lines: []tax.Set{{outsideScope}, {outsideScope}}, want: true},
		{name: "outside scope mixed with standard (BR-O-11)", lines: []tax.Set{{outsideScope}, {standard}}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invoice := new(bill.Invoice)
			for _, set := range test.lines {
				invoice.Lines = append(invoice.Lines, &bill.Line{Taxes: set})
			}

			if got := outsideScopeStandsAlone(invoice); got != test.want {
				t.Errorf("outside scope stands alone: want %v, got %v", test.want, got)
			}
		})
	}
}
