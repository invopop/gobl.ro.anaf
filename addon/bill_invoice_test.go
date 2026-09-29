package efactura

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func TestNormalizeInvoiceCustomer(t *testing.T) {
	t.Run("an invoice with no customer is left alone", func(t *testing.T) {
		invoice := new(bill.Invoice)

		normalizeInvoice(invoice)

		if invoice.Customer != nil {
			t.Errorf("customer: want none, got %v", invoice.Customer)
		}
	})

	t.Run("a romanian consumer gets the placeholder number", func(t *testing.T) {
		invoice := &bill.Invoice{
			Customer: &org.Party{
				Name: "Ion Popescu",
				Addresses: []*org.Address{
					{Street: "Strada Lunga", Locality: "Brasov", Country: l10n.RO.ISO()},
				},
			},
		}

		normalizeInvoice(invoice)

		identity := FindCNP(invoice.Customer)
		if identity == nil {
			t.Fatal("customer identity: want a placeholder CNP, got none")
		}
		if identity.Code != AnonymousCNP {
			t.Errorf("customer identity code: want %s, got %s", AnonymousCNP, identity.Code)
		}
		if identity.Scope != org.IdentityScopeLegal {
			t.Errorf("customer identity scope: want %s, got %s", org.IdentityScopeLegal, identity.Scope)
		}
	})

	t.Run("a consumer with no address is read as romanian", func(t *testing.T) {
		invoice := &bill.Invoice{
			Customer: &org.Party{Name: "Ion Popescu"},
		}

		normalizeInvoice(invoice)

		if FindCNP(invoice.Customer) == nil {
			t.Error("customer identity: want a placeholder CNP, got none")
		}
	})

	t.Run("a foreign consumer gets the placeholder number too", func(t *testing.T) {
		invoice := &bill.Invoice{
			Customer: &org.Party{
				Name: "John Smith",
				Addresses: []*org.Address{
					{Street: "Main St", Locality: "Austin", Country: l10n.US.ISO()},
				},
			},
		}

		normalizeInvoice(invoice)

		identity := FindCNP(invoice.Customer)
		if identity == nil {
			t.Fatal("customer identity: want a placeholder CNP, got none")
		}
		if identity.Code != AnonymousCNP {
			t.Errorf("customer identity code: want %s, got %s", AnonymousCNP, identity.Code)
		}
	})

	t.Run("consumer number is promoted into BT-47", func(t *testing.T) {
		invoice := &bill.Invoice{
			Customer: &org.Party{
				Name: "Ion Popescu",
				Identities: []*org.Identity{
					{Type: IdentityTypeCNP, Code: "1900101123456"},
				},
			},
		}

		normalizeInvoice(invoice)

		identity := FindCNP(invoice.Customer)
		if identity.Scope != org.IdentityScopeLegal {
			t.Errorf("customer identity scope: want %s, got %s", org.IdentityScopeLegal, identity.Scope)
		}
		if len(invoice.Customer.Identities) != 1 {
			t.Errorf("customer identities: want 1, got %d", len(invoice.Customer.Identities))
		}
	})

	t.Run("a business is left alone", func(t *testing.T) {
		invoice := &bill.Invoice{
			Customer: &org.Party{
				Name:  "Client Industrial SA",
				TaxID: &tax.Identity{Country: l10n.RO.Tax(), Code: cbc.Code("12345674")},
			},
		}

		normalizeInvoice(invoice)

		if len(invoice.Customer.Identities) != 0 {
			t.Errorf("customer identities: want none, got %d", len(invoice.Customer.Identities))
		}
	})
}

func TestNormalizeInvoiceRounding(t *testing.T) {
	invoice := new(bill.Invoice)

	normalizeInvoice(invoice)

	if invoice.Tax == nil {
		t.Fatal("tax: want a rounding rule, got none")
	}
	if invoice.Tax.Rounding != tax.RoundingRuleCurrency {
		t.Errorf("rounding: want %s, got %s", tax.RoundingRuleCurrency, invoice.Tax.Rounding)
	}
}
