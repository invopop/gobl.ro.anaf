package roefactura_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl.ro.anaf/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"

	roefactura "github.com/invopop/gobl.ro.anaf"
)

// The corpus that proves the converter against ANAF's own schematron lives in
// the test folder of this package. These tests cover only what this
// package decides on its own.

func TestConvertRejectsWhatANAFNeverReceives(t *testing.T) {
	tests := []struct {
		name    string
		kind    cbc.Key
		wantErr error
	}{
		{name: "proforma is never reported", kind: bill.InvoiceTypeProforma, wantErr: roefactura.ErrSkipped},
		{name: "debit note cannot be filed", kind: bill.InvoiceTypeDebitNote, wantErr: roefactura.ErrUnsupported},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invoice := newInvoice()
			invoice.Type = test.kind

			_, err := roefactura.ForwardConvert(envelope(t, invoice))

			if !errors.Is(err, test.wantErr) {
				t.Errorf("error: want %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestConvertRejectsAnEmptyEnvelope(t *testing.T) {
	_, err := roefactura.ForwardConvert(nil)

	if !errors.Is(err, roefactura.ErrInvalidDocument) {
		t.Errorf("error: want %v, got %v", roefactura.ErrInvalidDocument, err)
	}
}

func TestConvertRejectsAnInvoiceWithoutTheAddon(t *testing.T) {
	invoice := newInvoice()
	invoice.Addons = tax.Addons{}

	_, err := roefactura.ForwardConvert(envelope(t, invoice))

	if !errors.Is(err, roefactura.ErrInvalidDocument) {
		t.Errorf("error: want %v, got %v", roefactura.ErrInvalidDocument, err)
	}
}

func TestConvertWritesTheAccountingCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency currency.Code
		rates    []*currency.ExchangeRate
		wantTax  bool
	}{
		{name: "an invoice in RON states no BT-6", currency: currency.RON},
		{
			name:     "an invoice in EUR states BT-6 and BT-111",
			currency: currency.EUR,
			rates:    []*currency.ExchangeRate{{From: currency.EUR, To: currency.RON, Amount: num.MakeAmount(50850, 4)}},
			wantTax:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invoice := newInvoice()
			invoice.Currency = test.currency
			invoice.ExchangeRates = test.rates

			document, err := roefactura.ForwardConvert(envelope(t, invoice))
			if err != nil {
				t.Fatalf("conversion failed: %v", err)
			}

			got := strings.Contains(string(document), "<cbc:TaxCurrencyCode>RON</cbc:TaxCurrencyCode>")
			if got != test.wantTax {
				t.Errorf("BT-6 present: want %t, got %t", test.wantTax, got)
			}

			if test.wantTax && !strings.Contains(string(document), `currencyID="RON"`) {
				t.Error("BT-111 is missing, so EN 16931 BR-53 cannot hold")
			}
		})
	}
}

func TestConvertDropsWhatDoesNotBelongInACIUSRODocument(t *testing.T) {
	document, err := roefactura.ForwardConvert(envelope(t, newInvoice()))
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	rendered := string(document)

	if strings.Contains(rendered, "schemaLocation") {
		t.Error("the document still points at a schema location nobody can resolve")
	}

	if strings.Contains(rendered, "<cac:OrderReference>") {
		t.Error("the Peppol placeholder order reference reached a Romanian document")
	}
}

func TestConvertKeepsARealOrderReference(t *testing.T) {
	invoice := newInvoice()
	invoice.Ordering = &bill.Ordering{Purchases: []*org.DocumentRef{{Code: "NA"}}}

	document, err := roefactura.ForwardConvert(envelope(t, invoice))
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	if !strings.Contains(string(document), "<cac:OrderReference>") {
		t.Error("a purchase order the buyer really named NA was dropped as a placeholder")
	}
}

func TestConvertWritesTheSelfBillingMentionOnce(t *testing.T) {
	tests := []struct {
		name  string
		notes []*org.Note
	}{
		{name: "the mention is added to a buyer-issued invoice"},
		{name: "a mention the buyer already wrote is not repeated", notes: []*org.Note{{Text: "Autofactura emisa conform acordului"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invoice := newInvoice()
			invoice.SetTags(tax.TagSelfBilled)
			invoice.Notes = test.notes

			document, err := roefactura.ForwardConvert(envelope(t, invoice))
			if err != nil {
				t.Fatalf("conversion failed: %v", err)
			}

			if got := strings.Count(strings.ToLower(string(document)), "<cbc:note>autofactur"); got != 1 {
				t.Errorf("autofactura mentions in BT-22: want 1, got %d", got)
			}
		})
	}
}

func TestConvertKeepsTheSupplierCIFOutsideScope(t *testing.T) {
	// A seller not registered for VAT, with a tax representative as ordering seller
	invoice := newInvoice()
	invoice.Lines[0].Taxes = tax.Set{{
		Category: tax.CategoryVAT,
		Key:      tax.KeyOutsideScope,
		Ext:      tax.ExtensionsOf(cbc.CodeMap{"cef-vatex": "VATEX-EU-O"}),
	}}
	invoice.Ordering = &bill.Ordering{Seller: &org.Party{
		Name:  "Reprezentant Fiscal SRL",
		TaxID: &tax.Identity{Country: l10n.RO.Tax(), Code: "34567892"},
	}}

	document, err := roefactura.ForwardConvert(envelope(t, invoice))
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	// BT-30 and BT-32 belong to the supplier, never to its tax representative
	supplier := string(document)
	start := strings.Index(supplier, "<cac:AccountingSupplierParty>")
	end := strings.Index(supplier, "</cac:AccountingSupplierParty>")
	if start < 0 || end < start {
		t.Fatal("the document has no supplier party")
	}
	supplier = supplier[start:end]

	if !strings.Contains(supplier, ">12345674<") {
		t.Error("the supplier's own CIF is missing from its party")
	}
	if strings.Contains(supplier, ">34567892<") {
		t.Error("the tax representative's CIF was written as the supplier's")
	}
}

// newInvoice builds the smallest document Romania accepts.
func newInvoice() *bill.Invoice {
	address := &org.Address{
		Street: "Bulevardul Libertatii", Number: "16", Locality: "Sector 1",
		Region: "RO-B", Code: "010101", Country: l10n.RO.ISO(),
	}

	return &bill.Invoice{
		Addons:    tax.WithAddons(addon.V1),
		Type:      bill.InvoiceTypeStandard,
		Series:    "FCT",
		Code:      "2026-0001",
		IssueDate: cal.MakeDate(2026, 8, 28),
		Currency:  currency.RON,
		Supplier: &org.Party{
			Name:      "Exemplu Digital SRL",
			TaxID:     &tax.Identity{Country: l10n.RO.Tax(), Code: "12345674"},
			Addresses: []*org.Address{address},
		},
		Customer: &org.Party{
			Name:      "Client Industrial SA",
			TaxID:     &tax.Identity{Country: l10n.RO.Tax(), Code: "87654329"},
			Addresses: []*org.Address{address},
		},
		Lines: []*bill.Line{
			{
				Quantity: num.MakeAmount(1, 0),
				Item:     &org.Item{Name: "Serviciu", Price: num.NewAmount(10000, 2)},
				Taxes:    tax.Set{{Category: tax.CategoryVAT, Percent: num.NewPercentage(21, 2)}},
			},
		},
		Payment: &bill.PaymentDetails{
			Instructions: &pay.Instructions{Key: pay.MeansKeyCreditTransfer},
			Terms:        &pay.Terms{Notes: "Plata in 30 de zile"},
		},
	}
}

func envelope(t *testing.T, invoice *bill.Invoice) *gobl.Envelope {
	t.Helper()

	env, err := gobl.Envelop(invoice)
	if err != nil {
		t.Fatalf("cannot build an envelope: %v", err)
	}

	return env
}
