package roefactura_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"

	roefactura "github.com/invopop/gobl.ro.anaf"
	efactura "github.com/invopop/gobl.ro.anaf/addon"
)

func TestKindOf(t *testing.T) {
	tests := []struct {
		name    string
		invoice *bill.Invoice
		want    roefactura.Kind
	}{
		{
			name:    "no invoice",
			invoice: nil,
			want:    roefactura.KindOther,
		},
		{
			name:    "standard invoice",
			invoice: invoice(bill.InvoiceTypeStandard),
			want:    roefactura.KindInvoice,
		},
		{
			name:    "self billed invoice",
			invoice: invoice(bill.InvoiceTypeStandard, tax.TagSelfBilled),
			want:    roefactura.KindSelfBilled,
		},
		{
			name:    "accounting invoice",
			invoice: invoice(bill.InvoiceTypeStandard, efactura.TagAccounting),
			want:    roefactura.KindAccounting,
		},
		{
			name:    "self billing wins over accounting",
			invoice: invoice(bill.InvoiceTypeStandard, tax.TagSelfBilled, efactura.TagAccounting),
			want:    roefactura.KindSelfBilled,
		},
		{
			name:    "credit note",
			invoice: invoice(bill.InvoiceTypeCreditNote),
			want:    roefactura.KindCreditNote,
		},
		{
			name:    "self billed credit note is still a credit note",
			invoice: invoice(bill.InvoiceTypeCreditNote, tax.TagSelfBilled),
			want:    roefactura.KindCreditNote,
		},
		{
			name:    "corrective",
			invoice: invoice(bill.InvoiceTypeCorrective),
			want:    roefactura.KindCorrective,
		},
		{
			name:    "proforma is never filed",
			invoice: invoice(bill.InvoiceTypeProforma),
			want:    roefactura.KindProforma,
		},
		{
			name:    "debit note is unsupported",
			invoice: invoice(bill.InvoiceTypeDebitNote),
			want:    roefactura.KindDebitNote,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := roefactura.KindOf(test.invoice)

			if got != test.want {
				t.Errorf("kind: want %s, got %s", test.want.Name, got.Name)
			}
		})
	}
}

// invoice builds the smallest document KindOf reads.
func invoice(kind cbc.Key, tags ...cbc.Key) *bill.Invoice {
	out := &bill.Invoice{Type: kind}
	if len(tags) > 0 {
		out.Tags = tax.WithTags(tags...)
	}

	return out
}
