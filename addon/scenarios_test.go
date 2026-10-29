package efactura

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// selfBilledInvoice is a self-billed invoice from the supplier to the customer CIF.
func selfBilledInvoice(customerCIF cbc.Code) *bill.Invoice {
	return &bill.Invoice{
		Addons:    tax.WithAddons(V1),
		Tags:      tax.WithTags(tax.TagSelfBilled),
		Type:      bill.InvoiceTypeStandard,
		Code:      "AF-1",
		Currency:  "RON",
		IssueDate: cal.MakeDate(2025, 9, 1),
		Supplier: &org.Party{
			Name:  "Furnizor SRL",
			TaxID: &tax.Identity{Country: l10n.RO.Tax(), Code: "12345674"},
		},
		Customer: &org.Party{
			Name:  "Client SRL",
			TaxID: &tax.Identity{Country: l10n.RO.Tax(), Code: customerCIF},
		},
		Lines: []*bill.Line{
			{
				Quantity: num.MakeAmount(1, 0),
				Item:     &org.Item{Name: "Servicii", Price: num.NewAmount(10000, 2)},
				Taxes:    tax.Set{{Category: tax.CategoryVAT, Key: tax.KeyStandard, Percent: num.NewPercentage(21, 2)}},
			},
		},
	}
}

func TestSelfBillingScenarioNote(t *testing.T) {
	t.Run("a buyer-issued invoice carries the autofactura mention", func(t *testing.T) {
		invoice := selfBilledInvoice("87654329")
		if err := invoice.Calculate(); err != nil {
			t.Fatalf("calculate: %v", err)
		}

		if invoice.Tax == nil || len(invoice.Tax.Notes) != 1 || invoice.Tax.Notes[0].Text != SelfBillingNote {
			t.Errorf("tax notes: want the %q mention, got %v", SelfBillingNote, invoice.Tax.Notes)
		}
	})

	t.Run("a company billing a supply to itself carries no mention", func(t *testing.T) {
		invoice := selfBilledInvoice("12345674")
		if err := invoice.Calculate(); err != nil {
			t.Fatalf("calculate: %v", err)
		}

		if invoice.Tax != nil && len(invoice.Tax.Notes) != 0 {
			t.Errorf("tax notes: want none, got %v", invoice.Tax.Notes)
		}
	})
}
