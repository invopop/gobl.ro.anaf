package roefactura_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"

	roefactura "github.com/invopop/gobl.ro.anaf"
	efactura "github.com/invopop/gobl.ro.anaf/addon"
)

func TestCIF(t *testing.T) {
	tests := []struct {
		name  string
		party *org.Party
		want  string
	}{
		{
			name:  "tax number without the RO attribute",
			party: &org.Party{TaxID: &tax.Identity{Country: l10n.RO.Tax(), Code: cbc.Code("RO12345674")}},
			want:  "12345674",
		},
		{
			name:  "enforcement body named by its legal registration identifier (BR-RO-130)",
			party: &org.Party{Identities: []*org.Identity{{Scope: org.IdentityScopeLegal, Code: "22222224"}}},
			want:  "22222224",
		},
		{
			name:  "consumer named by a personal number",
			party: &org.Party{Identities: []*org.Identity{{Scope: org.IdentityScopeLegal, Type: efactura.IdentityTypeCNP, Code: efactura.AnonymousCNP}}},
			want:  "",
		},
		{
			name:  "no party",
			party: nil,
			want:  "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := roefactura.CIF(test.party); got != test.want {
				t.Errorf("cif: want %q, got %q", test.want, got)
			}
		})
	}
}

func TestIssuerOfAnEnforcementInvoice(t *testing.T) {
	payee := &org.Party{Name: "Birou Executor Judecatoresc Popescu", Identities: []*org.Identity{{Scope: org.IdentityScopeLegal, Code: "22222224"}}}
	invoice := &bill.Invoice{Tags: tax.WithTags(efactura.TagEnforcement), Payment: &bill.PaymentDetails{Payee: payee}}

	if got := roefactura.CIF(roefactura.Issuer(invoice)); got != "22222224" {
		t.Errorf("filing cif: want 22222224, got %q", got)
	}
}
