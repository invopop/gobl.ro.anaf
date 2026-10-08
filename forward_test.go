package roefactura_test

import (
	"bytes"
	"encoding/json"
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

func TestConvertReportsAnUncompliantDocument(t *testing.T) {
	invoice := newInvoice()
	invoice.Code = "FARA-CIFRE"

	_, err := roefactura.ForwardConvert(envelope(t, invoice))

	if !errors.Is(err, roefactura.ErrNotCompliant) {
		t.Fatalf("error: want %v, got %v", roefactura.ErrNotCompliant, err)
	}

	if !strings.Contains(err.Error(), "BILL-INVOICE-01") {
		t.Errorf("error should name the rule that rejected it, got %v", err)
	}
}

func TestConvertLeavesTheCallersEnvelopeAlone(t *testing.T) {
	// The send action reads the silo entry again, so conversion has to leave
	// the document exactly as it arrived, uncalculated and unnormalized.
	env := silo(t, newInvoice())

	before, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("cannot read the envelope: %v", err)
	}

	if _, err := roefactura.ForwardConvert(env); err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	after, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("cannot read the envelope: %v", err)
	}

	if !bytes.Equal(before, after) {
		t.Error("conversion rewrote the envelope the caller handed over")
	}

	invoice, ok := env.Extract().(*bill.Invoice)
	if !ok {
		t.Fatal("envelope carries no invoice")
	}

	if invoice.Totals != nil {
		t.Error("conversion calculated the caller's invoice in place")
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

// silo returns the envelope the way a silo entry reaches the converter: parsed
// from JSON, so nothing has calculated or normalized it yet.
func silo(t *testing.T, invoice *bill.Invoice) *gobl.Envelope {
	t.Helper()

	payload, err := json.Marshal(invoice)
	if err != nil {
		t.Fatalf("cannot build a silo entry: %v", err)
	}

	doc := make(map[string]any)
	if err := json.Unmarshal(payload, &doc); err != nil {
		t.Fatalf("cannot build a silo entry: %v", err)
	}
	doc["$schema"] = "https://gobl.org/draft-0/bill/invoice"

	data, err := json.Marshal(map[string]any{
		"$schema": "https://gobl.org/draft-0/envelope",
		"head":    map[string]any{"uuid": "0195ce71-dc9c-72c8-bf2c-9890a4a9f100"},
		"doc":     doc,
	})
	if err != nil {
		t.Fatalf("cannot build a silo entry: %v", err)
	}

	env := new(gobl.Envelope)
	if err := json.Unmarshal(data, env); err != nil {
		t.Fatalf("cannot read a silo entry: %v", err)
	}

	return env
}

func envelope(t *testing.T, invoice *bill.Invoice) *gobl.Envelope {
	t.Helper()

	env, err := gobl.Envelop(invoice)
	if err != nil {
		t.Fatalf("cannot build an envelope: %v", err)
	}

	return env
}
