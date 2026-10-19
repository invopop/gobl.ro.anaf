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

	t.Run("a foreign buyer gets no placeholder number", func(t *testing.T) {
		invoice := &bill.Invoice{
			Customer: &org.Party{
				Name: "Acme Inc",
				Addresses: []*org.Address{
					{Street: "Main St", Locality: "Austin", Country: l10n.US.ISO()},
				},
			},
		}

		normalizeInvoice(invoice)

		if len(invoice.Customer.Identities) != 0 {
			t.Errorf("customer identities: want none, got %v", invoice.Customer.Identities)
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

func TestSellerIdentified(t *testing.T) {
	romanian := &tax.Identity{Country: l10n.RO.Tax(), Code: cbc.Code("12345674")}

	tests := []struct {
		name    string
		invoice *bill.Invoice
		want    bool
	}{
		{
			name:    "supplier with a VAT identifier (BT-31)",
			invoice: &bill.Invoice{Supplier: &org.Party{TaxID: romanian}},
			want:    true,
		},
		{
			name: "supplier with a tax registration identifier (BT-32)",
			invoice: &bill.Invoice{Supplier: &org.Party{Identities: []*org.Identity{
				{Scope: org.IdentityScopeTax, Code: "12345674"},
			}}},
			want: true,
		},
		{
			name: "tax representative with a VAT identifier (BT-63)",
			invoice: &bill.Invoice{
				Supplier: &org.Party{Name: "Seller"},
				Ordering: &bill.Ordering{Seller: &org.Party{TaxID: romanian}},
			},
			want: true,
		},
		{
			name:    "no identifier at all",
			invoice: &bill.Invoice{Supplier: &org.Party{Name: "Seller"}},
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sellerIdentified(test.invoice); got != test.want {
				t.Errorf("seller identified: want %v, got %v", test.want, got)
			}
		})
	}
}

func TestEnforcementBodyIdentified(t *testing.T) {
	enforcement := func(payee *org.Party) *bill.Invoice {
		invoice := &bill.Invoice{Tags: tax.WithTags(TagEnforcement)}
		if payee != nil {
			invoice.Payment = &bill.PaymentDetails{Payee: payee}
		}
		return invoice
	}

	tests := []struct {
		name    string
		invoice *bill.Invoice
		want    bool
	}{
		{
			name: "payee with a name and a legal registration identifier",
			invoice: enforcement(&org.Party{
				Name:       "Birou Executor Judecatoresc Popescu",
				Identities: []*org.Identity{{Scope: org.IdentityScopeLegal, Code: "22222224"}},
			}),
			want: true,
		},
		{
			name: "payee with a tax identifier only, which is not BT-61",
			invoice: enforcement(&org.Party{
				Name:  "Birou Executor Judecatoresc Popescu",
				TaxID: &tax.Identity{Country: l10n.RO.Tax(), Code: cbc.Code("22222224")},
			}),
			want: false,
		},
		{
			name: "payee with no name",
			invoice: enforcement(&org.Party{
				Identities: []*org.Identity{{Scope: org.IdentityScopeLegal, Code: "22222224"}},
			}),
			want: false,
		},
		{
			name:    "no payee",
			invoice: enforcement(nil),
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := enforcementBodyIdentified(test.invoice); got != test.want {
				t.Errorf("enforcement body identified: want %v, got %v", test.want, got)
			}
		})
	}
}
