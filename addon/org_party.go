package efactura

import (
	"strings"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

// Lengths the RO_CIUS puts on party fields, BR-RO-L100 and BR-RO-L200.
const (
	maxPartyName    = 200
	maxContactName  = 100
	maxContactPhone = 100
	maxContactEmail = 100
)

// IsBusiness reports a party with a tax number or a non-CNP legal identity. A
// party carrying only a personal number is a consumer, which picks the endpoint.
func IsBusiness(party *org.Party) bool {
	if party == nil {
		return false
	}

	if HasTaxNumber(party) {
		return true
	}

	for _, identity := range party.Identities {
		if identity == nil || identity.Scope != org.IdentityScopeLegal {
			continue
		}
		if cbc.NormalizeUpperCode(identity.Type) != IdentityTypeCNP {
			return true
		}
	}

	return false
}

// HasTaxNumber reports a party carrying a tax identity with a code.
func HasTaxNumber(party *org.Party) bool {
	return party != nil && party.TaxID != nil && party.TaxID.Code != ""
}

// CIF returns the party's tax number bare, without the RO fiscal attribute.
func CIF(party *org.Party) string {
	if !HasTaxNumber(party) {
		return ""
	}

	code := strings.ToUpper(strings.TrimSpace(party.TaxID.Code.String()))
	if party.TaxID.Country.Code() == l10n.RO {
		code = strings.TrimPrefix(code, l10n.RO.String())
	}

	return code
}

// SameTaxNumber reports two parties registered under the same tax number.
func SameTaxNumber(a, b *org.Party) bool {
	if !HasTaxNumber(a) || !HasTaxNumber(b) {
		return false
	}

	return a.TaxID.Country == b.TaxID.Country && CIF(a) == CIF(b)
}

// HasLegalIdentity reports an identity GOBL will map to BT-47.
func HasLegalIdentity(party *org.Party) bool {
	if party == nil {
		return false
	}

	for _, identity := range party.Identities {
		if identity != nil && identity.Scope == org.IdentityScopeLegal {
			return true
		}
	}

	return false
}

// IsIdentified reports a party ANAF can attach the document to, by tax number
// or by the BT-47 legal identifier.
func IsIdentified(party *org.Party) bool {
	return HasTaxNumber(party) || HasLegalIdentity(party)
}

func orgPartyRules() *rules.Set {
	return rules.For(new(org.Party),
		rules.Field("name",
			rules.Assert("01", "party name (BT-27, BT-44, BT-59, BT-62, BT-70) must be no more than 200 characters long (BR-RO-L200)",
				is.RuneLength(0, maxPartyName),
			),
		),
		rules.Field("alias",
			rules.Assert("02", "party trading name (BT-28, BT-45) must be no more than 200 characters long (BR-RO-L200)",
				is.RuneLength(0, maxPartyName),
			),
		),
		rules.Field("people",
			rules.Assert("03", "contact point (BT-41, BT-56) must be no more than 100 characters long (BR-RO-L100)",
				is.Func("contact name within length", contactNameWithinLength),
			),
		),
		rules.Field("telephones",
			rules.Assert("04", "contact telephone number (BT-42, BT-57) must be no more than 100 characters long (BR-RO-L100)",
				is.Func("telephone within length", telephoneWithinLength),
			),
		),
		rules.Field("emails",
			rules.Assert("05", "contact email address (BT-43, BT-58) must be no more than 100 characters long (BR-RO-L100)",
				is.Func("email within length", emailWithinLength),
			),
		),
		rules.Field("addresses",
			rules.Assert("06", "address line 1 (BT-35, BT-50, BT-64, BT-75) must be no more than 150 characters long (BR-RO-L150)",
				is.Func("line one within length", addressLineOneWithinLength),
			),
			rules.Assert("07", "address line 2 (BT-36, BT-51, BT-65, BT-76) must be no more than 100 characters long (BR-RO-L100)",
				is.Func("line two within length", addressLineTwoWithinLength),
			),
			rules.Assert("08", "city (BT-37, BT-52, BT-66, BT-77) must be no more than 50 characters long (BR-RO-L050)",
				is.Func("locality within length", addressLocalityWithinLength),
			),
			rules.Assert("09", "post code (BT-38, BT-53, BT-67, BT-78) must be no more than 20 characters long (BR-RO-L020)",
				is.Func("post code within length", addressPostCodeWithinLength),
			),
			rules.Assert("10", "country subdivision (BT-39, BT-54, BT-68, BT-79) must be an ISO 3166-2:RO code, such as RO-B or RO-AB (BR-RO-110, BR-RO-170, BR-RO-210)",
				is.Func("ISO 3166-2:RO subdivision", addressRegionIsSubentity),
			),
			rules.Assert("11", "city (BT-37, BT-52, BT-66, BT-77) must be a SECTOR-RO code, such as SECTOR1, when the country subdivision is RO-B (BR-RO-100, BR-RO-160, BR-RO-200)",
				is.Func("SECTOR-RO code", addressLocalityIsSector),
			),
		),
	)
}

func contactNameWithinLength(value any) bool {
	people, ok := value.([]*org.Person)
	if !ok || len(people) == 0 || people[0] == nil {
		return true
	}

	return len([]rune(contactName(people[0].Name))) <= maxContactName
}

// contactName renders BT-41 the way gobl.ubl does.
func contactName(name *org.Name) string {
	if name == nil {
		return ""
	}

	parts := make([]string, 0, 2)
	for _, part := range []string{name.Given, name.Surname} {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, " ")
}

func telephoneWithinLength(value any) bool {
	telephones, ok := value.([]*org.Telephone)
	if !ok || len(telephones) == 0 || telephones[0] == nil {
		return true
	}

	return len([]rune(telephones[0].Number)) <= maxContactPhone
}

func emailWithinLength(value any) bool {
	emails, ok := value.([]*org.Email)
	if !ok || len(emails) == 0 || emails[0] == nil {
		return true
	}

	return len([]rune(emails[0].Address)) <= maxContactEmail
}

func addressLineOneWithinLength(value any) bool {
	address := addressesExported(value)

	return address == nil || len([]rune(address.LineOne())) <= maxAddressLineOne
}

func addressLineTwoWithinLength(value any) bool {
	address := addressesExported(value)

	return address == nil || len([]rune(address.LineTwo())) <= maxAddressLineTwo
}

func addressLocalityWithinLength(value any) bool {
	address := addressesExported(value)

	return address == nil || len([]rune(address.Locality)) <= maxAddressLocality
}

func addressPostCodeWithinLength(value any) bool {
	address := addressesExported(value)

	return address == nil || len([]rune(address.Code.String())) <= maxAddressPostCode
}

func addressRegionIsSubentity(value any) bool {
	address := addressesExported(value)
	if !addressInRomania(address) {
		return true
	}

	return IsSubentity(address.Region)
}

func addressLocalityIsSector(value any) bool {
	address := addressesExported(value)
	if !addressInRomania(address) || address.Region != Subentity(CountyBucharest) {
		return true
	}

	return IsSector(address.Locality)
}
