package efactura

import (
	"strings"
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// The rule tests are handed whatever value sits at their path, so each one must
// cope with a missing or unexpected value without panicking.
func TestRuleTestsWithoutAValue(t *testing.T) {
	tests := []struct {
		name string
		test func(any) bool
		want bool
	}{
		{name: "invoice number has a digit", test: invoiceNumberHasDigit, want: true},
		{name: "invoice number within length", test: invoiceNumberWithinLength, want: true},
		{name: "notes within count", test: notesWithinCount, want: true},
		{name: "currency within subunits", test: currencyWithinSubunits, want: true},
		{name: "preceding numbers within length", test: precedingNumbersWithinLength, want: true},
		{name: "exemption reasons within length", test: exemptionReasonsWithinLength, want: true},
		{name: "outside scope stands alone", test: outsideScopeStandsAlone, want: true},
		{name: "seller identified", test: sellerIdentified, want: true},
		{name: "enforcement body identified", test: enforcementBodyIdentified, want: true},
		{name: "account name within length", test: accountNameWithinLength, want: true},
		{name: "attribute value within length", test: attributeValueWithinLength, want: true},
		{name: "note within length", test: noteWithinLength, want: true},
		{name: "accounting reference within length", test: accountingRefWithinLength, want: true},
		{name: "party address has a street", test: partyAddressHasStreet, want: true},
		{name: "party address has a city", test: partyAddressHasLocality, want: true},
		{name: "contact name within length", test: contactNameWithinLength, want: true},
		{name: "telephone within length", test: telephoneWithinLength, want: true},
		{name: "email within length", test: emailWithinLength, want: true},
		{name: "address line one within length", test: addressLineOneWithinLength, want: true},
		{name: "address line two within length", test: addressLineTwoWithinLength, want: true},
		{name: "address city within length", test: addressLocalityWithinLength, want: true},
		{name: "address post code within length", test: addressPostCodeWithinLength, want: true},
		{name: "address region is a subentity", test: addressRegionIsSubentity, want: true},
		{name: "address city is a sector", test: addressLocalityIsSector, want: true},
		{name: "customer identified", test: customerIdentified, want: false},
		{name: "invoice is an enforcement", test: invoiceIsEnforcement, want: false},
		{name: "delivery has an address", test: deliveryHasAddress, want: false},
		{name: "delivery address has a street", test: deliveryAddressHasStreet, want: false},
		{name: "delivery address has a city", test: deliveryAddressHasLocality, want: false},
		{name: "delivery address has a region", test: deliveryAddressHasRegion, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, value := range []any{nil, "not the expected type"} {
				if got := test.test(value); got != test.want {
					t.Errorf("%T value: want %v, got %v", value, test.want, got)
				}
			}
		})
	}
}

func TestInvoiceNumberRules(t *testing.T) {
	tests := []struct {
		name       string
		series     cbc.Code
		code       cbc.Code
		wantDigit  bool
		wantLength bool
	}{
		{name: "series and code with a digit", series: "FCT", code: "2026-0001", wantDigit: true, wantLength: true},
		{name: "code alone with a digit", code: "A1", wantDigit: true, wantLength: true},
		{name: "no digit anywhere (BR-RO-010)", series: "FCT", code: "ABC", wantDigit: false, wantLength: true},
		{name: "longer than 200 characters (BR-RO-L200)", code: cbc.Code("1" + strings.Repeat("A", 200)), wantDigit: true, wantLength: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invoice := &bill.Invoice{Series: test.series, Code: test.code}

			if got := invoiceNumberHasDigit(invoice); got != test.wantDigit {
				t.Errorf("has a digit: want %v, got %v", test.wantDigit, got)
			}
			if got := invoiceNumberWithinLength(invoice); got != test.wantLength {
				t.Errorf("within length: want %v, got %v", test.wantLength, got)
			}
		})
	}
}

func TestCurrencyWithinSubunits(t *testing.T) {
	tests := []struct {
		name     string
		currency currency.Code
		want     bool
	}{
		{name: "RON has two decimals", currency: currency.RON, want: true},
		{name: "no currency yet", currency: currency.CodeEmpty, want: true},
		{name: "KWD has three decimals (BR-RO-Z2)", currency: currency.KWD, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := currencyWithinSubunits(&bill.Invoice{Currency: test.currency}); got != test.want {
				t.Errorf("within subunits: want %v, got %v", test.want, got)
			}
		})
	}
}

func TestPrecedingNumbersWithinLength(t *testing.T) {
	short := &org.DocumentRef{Series: "FCT", Code: "2026-0001"}
	long := &org.DocumentRef{Code: cbc.Code(strings.Repeat("1", 201))}

	if !precedingNumbersWithinLength([]*org.DocumentRef{nil, short}) {
		t.Error("a short preceding number must pass, skipping empty entries")
	}
	if precedingNumbersWithinLength([]*org.DocumentRef{short, long}) {
		t.Error("a preceding number longer than 200 characters must fail (BR-RO-L200)")
	}
}

func TestAttributeValue(t *testing.T) {
	tests := []struct {
		name      string
		attribute *org.Attribute
		want      string
	}{
		{name: "no attribute", attribute: nil, want: ""},
		{name: "plain text", attribute: &org.Attribute{Label: "Culoare", Text: "Rosu"}, want: "Rosu"},
		{name: "amount with its unit", attribute: &org.Attribute{Label: "Greutate", Amount: num.NewAmount(125, 1), Unit: org.UnitKilogram}, want: "12.5 kg"},
		{name: "amount alone", attribute: &org.Attribute{Label: "Bucati", Amount: num.NewAmount(3, 0)}, want: "3"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := attributeValue(test.attribute); got != test.want {
				t.Errorf("value: want %q, got %q", test.want, got)
			}
		})
	}
}

func TestIsBusiness(t *testing.T) {
	romanian := &tax.Identity{Country: l10n.RO.Tax(), Code: "12345674"}

	tests := []struct {
		name  string
		party *org.Party
		want  bool
	}{
		{name: "no party", party: nil, want: false},
		{name: "a tax number", party: &org.Party{TaxID: romanian}, want: true},
		{
			name:  "a legal identity other than a personal number",
			party: &org.Party{Identities: []*org.Identity{{Scope: org.IdentityScopeLegal, Type: "ONRC", Code: "J40/1234/2020"}}},
			want:  true,
		},
		{
			name:  "only a personal number",
			party: &org.Party{Identities: []*org.Identity{{Scope: org.IdentityScopeLegal, Type: IdentityTypeCNP, Code: AnonymousCNP}}},
			want:  false,
		},
		{
			name:  "an identity outside the legal scope",
			party: &org.Party{Identities: []*org.Identity{nil, {Scope: org.IdentityScopeTax, Code: "12345674"}}},
			want:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsBusiness(test.party); got != test.want {
				t.Errorf("is business: want %v, got %v", test.want, got)
			}
		})
	}
}

func TestSameTaxNumber(t *testing.T) {
	party := func(country l10n.TaxCountryCode, code cbc.Code) *org.Party {
		return &org.Party{TaxID: &tax.Identity{Country: country, Code: code}}
	}

	tests := []struct {
		name string
		a, b *org.Party
		want bool
	}{
		{name: "the same CIF, one with the RO attribute", a: party(l10n.RO.Tax(), "RO12345674"), b: party(l10n.RO.Tax(), "12345674"), want: true},
		{name: "different CIFs", a: party(l10n.RO.Tax(), "12345674"), b: party(l10n.RO.Tax(), "87654329"), want: false},
		{name: "the same code in different countries", a: party(l10n.RO.Tax(), "12345674"), b: party(l10n.BG.Tax(), "12345674"), want: false},
		{name: "one party without a tax number", a: party(l10n.RO.Tax(), "12345674"), b: &org.Party{}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := SameTaxNumber(test.a, test.b); got != test.want {
				t.Errorf("same tax number: want %v, got %v", test.want, got)
			}
		})
	}
}
