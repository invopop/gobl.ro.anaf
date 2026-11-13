// Rules for every party, address, item, note and attachment of the invoice.
//
// Official sources:
//
//   - Order MF 4092/2022: the BR-RO table of national rules (art. 3 alin. (2))
//     https://static.anaf.ro/static/10/Anaf/legislatie/OMF_4092_2022.pdf
//   - ISO 3166-2:RO: the county codes listed in counties.go
//   - OUG 120/2021 art. 10^1 alin. (3), as amended by OUG 138/2024: a consumer who gives no
//     tax code is invoiced under a code of thirteen zeros
//     https://static.anaf.ro/static/10/Anaf/legislatie/OUG_138_2024.pdf
//
// Each rule names the official rule it applies at the end of its message, such as
// (BR-RO-010). The addon README maps every BR-RO rule to the fault code GOBL reports.

package efactura

import (
	"fmt"
	"strings"

	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

// IdentityTypeCNP is the Romanian personal numeric code, BT-47 of a consumer.
const IdentityTypeCNP cbc.Code = "CNP"

// AnonymousCNP is the code of thirteen zeros OUG 120/2021 art. 10^1 alin. (3) gives a consumer with no tax code.
const AnonymousCNP cbc.Code = "0000000000000"

// NoteKeyBuyerAccountingRef routes a line note into BT-133 rather than BT-127,
// which is why it takes the tighter limit.
const NoteKeyBuyerAccountingRef cbc.Key = "buyer-accounting-ref"

// Limits BR-RO-A050, L020, L050, L100, L150, L200 and L300 put on parties,
// addresses, items, notes and attachments.
const (
	maxPartyName             = 200
	maxLegalInfo             = 1000
	maxLineNotes             = 1
	maxContactName           = 100
	maxContactPhone          = 100
	maxContactEmail          = 100
	maxAddressLineOne        = 150
	maxAddressLineTwo        = 100
	maxAddressLocality       = 50
	maxAddressPostCode       = 20
	maxItemName              = 100
	maxItemDescription       = 200
	maxItemAttributes        = 50
	maxAttributeLabel        = 50
	maxAttributeValue        = 100
	maxNoteText              = 300
	maxNoteAccountingRef     = 100
	maxAttachmentDescription = 100
	maxAttachmentURL         = 200
	maxAttachmentName        = 200
)

// attributeUnitSeparate joins the amount and the unit of BT-161.
const attributeUnitSeparate = " "

var identities = []*cbc.Definition{
	{
		Code: IdentityTypeCNP,
		Name: i18n.String{
			i18n.EN: "Personal Numeric Code",
			i18n.RO: "Cod Numeric Personal",
		},
		Desc: i18n.String{
			i18n.EN: "Thirteen digit code identifying a Romanian natural person. Use thirteen zeros when the buyer of a consumer invoice gives no code.",
			i18n.RO: "Cod de treisprezece cifre care identifica o persoana fizica. Se folosesc treisprezece zerouri cand cumparatorul persoana fizica nu il comunica.",
		},
	},
}

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

// HasLegalIdentity reports an identity GOBL maps to the legal registration identifier, BT-30, BT-47 or BT-61.
func HasLegalIdentity(party *org.Party) bool {
	return LegalIdentity(party) != nil
}

// LegalIdentity returns the first legal identity, the one gobl.ubl writes as the legal registration identifier.
func LegalIdentity(party *org.Party) *org.Identity {
	if party == nil {
		return nil
	}

	for _, identity := range party.Identities {
		if identity != nil && identity.Scope == org.IdentityScopeLegal && identity.Code != "" {
			return identity
		}
	}

	return nil
}

// IsIdentified reports a party ANAF can attach the document to, by tax number
// or by the BT-47 legal identifier.
func IsIdentified(party *org.Party) bool {
	return HasTaxNumber(party) || HasLegalIdentity(party)
}

// hasTaxRegistration reports a tax identity GOBL maps to the tax registration identifier, BT-32.
func hasTaxRegistration(party *org.Party) bool {
	if party == nil {
		return false
	}

	for _, identity := range party.Identities {
		if identity != nil && identity.Scope == org.IdentityScopeTax && identity.Code != "" {
			return true
		}
	}

	return false
}

// FindCNP returns the party's personal number, normalizing the identity type.
func FindCNP(party *org.Party) *org.Identity {
	if party == nil {
		return nil
	}

	for _, identity := range party.Identities {
		if identity != nil && cbc.NormalizeUpperCode(identity.Type) == IdentityTypeCNP {
			return identity
		}
	}

	return nil
}

// The CNP, NIF and CUI checks below mirror ANAF's identifier check (ERRIdentif),
// which its validator runs next to the schematron. Source: class
// ro.gov.mfinante.utlis.ValidareCuiVanzatorCumparator (checkCNP, checkNIF, checkCUI)
// of ANAF's offline validator v1.3.0 (12.12.2024), published on
// https://mfinante.gov.ro/ro/web/efactura/informatii-tehnice. Weights, centuries
// and check digit rounding follow it exactly.

// personalNumberWeights is the key ANAF checks a CNP or NIF against.
var personalNumberWeights = []int{2, 7, 9, 1, 4, 6, 3, 5, 8, 2, 7, 9}

// cuiWeights is the key ANAF checks a CUI against, over the code padded to nine digits.
var cuiWeights = []int{7, 5, 3, 2, 1, 7, 5, 3, 2}

// monthDays is the longest each month runs, February counted in a leap year.
var monthDays = []int{31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// validPersonalNumber applies ANAF's CNP and NIF check (ERRIdentif): thirteen digits with a valid check digit, a NIF starting with 9, a CNP with a real birth date, or thirteen zeros.
func validPersonalNumber(code string) bool {
	if code == AnonymousCNP.String() {
		return true
	}

	digits, ok := digitsOf(code)
	if !ok || len(digits) != 13 || digits[0] == 0 {
		return false
	}

	// A CNP carries the birth date, the century coming from the first digit
	if digits[0] != 9 {
		century := 1900
		switch digits[0] {
		case 3, 4:
			century = 1800
		case 5, 6:
			century = 2000
		}
		year := century + digits[1]*10 + digits[2]
		month := digits[3]*10 + digits[4]
		day := digits[5]*10 + digits[6]

		if month < 1 || month > 12 || day < 1 || day > monthDays[month-1] {
			return false
		}
		if month == 2 && day == 29 && (year%4 != 0 || (year%100 == 0 && year%400 != 0)) {
			return false
		}
	}

	sum := 0
	for i, weight := range personalNumberWeights {
		sum += digits[i] * weight
	}
	check := sum % 11
	if check == 10 {
		check = 1
	}

	return check == digits[12]
}

// validCUI applies ANAF's CUI check (ERRIdentif): up to ten digits, no leading zero, and a valid check digit.
func validCUI(code string) bool {
	digits, ok := digitsOf(code)
	if !ok || len(digits) == 0 || len(digits) > 10 || digits[0] == 0 {
		return false
	}

	padded := make([]int, 10-len(digits), 10)
	padded = append(padded, digits...)

	sum := 0
	for i, weight := range cuiWeights {
		sum += padded[i] * weight
	}
	check := sum * 10 % 11
	if check == 10 {
		check = 0
	}

	return check == padded[9]
}

// digitsOf returns the code's digits, or false when it holds anything else.
func digitsOf(code string) ([]int, bool) {
	digits := make([]int, 0, len(code))
	for _, r := range code {
		if r < '0' || r > '9' {
			return nil, false
		}
		digits = append(digits, int(r-'0'))
	}

	return digits, true
}

// anafBuyerCode returns the buyer identifier ANAF checks without a VAT number: BT-48's tax registration, else BT-47, bare of the RO prefix, empty when neither is numeric.
func anafBuyerCode(party *org.Party) string {
	codes := make([]string, 0, 2)
	for _, scope := range []cbc.Key{org.IdentityScopeTax, org.IdentityScopeLegal} {
		for _, identity := range party.Identities {
			if identity != nil && identity.Scope == scope && identity.Code != "" {
				codes = append(codes, identity.Code.String())
				break
			}
		}
	}

	for _, code := range codes {
		code = strings.TrimSpace(code)
		if rest, ok := strings.CutPrefix(code, l10n.RO.String()); ok {
			return strings.TrimSpace(rest)
		}
		if _, numeric := digitsOf(code); numeric {
			return code
		}
	}

	return ""
}

// normalizeAddress rewrites a Romanian address into the ISO 3166-2:RO county
// and, in Bucharest, the sector code BR-RO-100 and BR-RO-110 ask for.
func normalizeAddress(address *org.Address) {
	if address.Country != l10n.RO.ISO() {
		return
	}

	county, ok := ParseCounty(address.Region, address.State.String())
	if !ok {
		return
	}

	address.Region = Subentity(county)

	if county != CountyBucharest {
		return
	}

	if sector, ok := ParseSector(address.Locality); ok {
		address.Locality = Sector(sector)
	}
}

func orgPartyRules() *rules.Set {
	return rules.For(new(org.Party),
		rules.Field("name",
			rules.Assert("01", "party name must be no more than 200 characters long (BR-RO-L200)",
				is.RuneLength(0, maxPartyName),
			),
		),
		rules.Field("alias",
			rules.Assert("02", "party trading name must be no more than 200 characters long (BR-RO-L200)",
				is.RuneLength(0, maxPartyName),
			),
		),
		rules.Field("people",
			rules.Assert("03", "contact point must be no more than 100 characters long (BR-RO-L100)",
				is.Func("contact name within length", contactNameWithinLength),
			),
		),
		rules.Field("telephones",
			rules.Assert("04", "contact telephone number must be no more than 100 characters long (BR-RO-L100)",
				is.Func("telephone within length", telephoneWithinLength),
			),
		),
		rules.Field("emails",
			rules.Assert("05", "contact email address must be no more than 100 characters long (BR-RO-L100)",
				is.Func("email within length", emailWithinLength),
			),
		),
		rules.Field("addresses",
			rules.Assert("06", "address line 1 must be no more than 150 characters long (BR-RO-L150)",
				is.Func("line one within length", addressLineOneWithinLength),
			),
			rules.Assert("07", "address line 2 must be no more than 100 characters long (BR-RO-L100)",
				is.Func("line two within length", addressLineTwoWithinLength),
			),
			rules.Assert("08", "city must be no more than 50 characters long (BR-RO-L050)",
				is.Func("locality within length", addressLocalityWithinLength),
			),
			rules.Assert("09", "post code must be no more than 20 characters long (BR-RO-L020)",
				is.Func("post code within length", addressPostCodeWithinLength),
			),
			rules.Assert("10", "country subdivision must be an ISO 3166-2:RO code, such as RO-B or RO-AB (BR-RO-110, BR-RO-170, BR-RO-210)",
				is.Func("ISO 3166-2:RO subdivision", addressRegionIsSubentity),
			),
			rules.Assert("11", "city must be a Bucharest sector, such as SECTOR1, when the country subdivision is RO-B (BR-RO-100, BR-RO-160, BR-RO-200)",
				is.Func("SECTOR-RO code", addressLocalityIsSector),
			),
		),
	)
}

func orgItemRules() *rules.Set {
	return rules.For(new(org.Item),
		rules.Field("name",
			rules.Assert("01", "item name must be no more than 100 characters long (BR-RO-L100)",
				is.RuneLength(0, maxItemName),
			),
		),
		rules.Field("description",
			rules.Assert("02", "item description must be no more than 200 characters long (BR-RO-L200)",
				is.RuneLength(0, maxItemDescription),
			),
		),
		rules.Field("attributes",
			rules.Assert("03", "item attributes must contain no more than 50 entries (BR-RO-A050)",
				is.Length(0, maxItemAttributes),
			),
		),
	)
}

func orgAttributeRules() *rules.Set {
	return rules.For(new(org.Attribute),
		rules.Field("label",
			rules.Assert("01", "item attribute name must be no more than 50 characters long (BR-RO-L050)",
				is.RuneLength(0, maxAttributeLabel),
			),
		),
		rules.Assert("02", "item attribute value must be no more than 100 characters long (BR-RO-L100)",
			is.Func("attribute value within length", attributeValueWithinLength),
		),
	)
}

func orgNoteRules() *rules.Set {
	return rules.For(new(org.Note),
		rules.Assert("01", "note, with its subject qualifier when set, must be no more than 300 characters long (BR-RO-L300)",
			is.Func("note within length", noteWithinLength),
		),
		rules.Assert("02", "invoice line buyer accounting reference must be no more than 100 characters long (BR-RO-L100)",
			is.Func("accounting reference within length", accountingRefWithinLength),
		),
	)
}

func orgAttachmentRules() *rules.Set {
	return rules.For(new(org.Attachment),
		rules.Field("description",
			rules.Assert("01", "supporting document description must be no more than 100 characters long (BR-RO-L100)",
				is.RuneLength(0, maxAttachmentDescription),
			),
		),
		rules.Field("url",
			rules.Assert("02", "external document location must be no more than 200 characters long (BR-RO-L200)",
				is.RuneLength(0, maxAttachmentURL),
			),
		),
		rules.Field("name",
			rules.Assert("03", "attached document filename must be no more than 200 characters long (BR-RO-L200)",
				is.RuneLength(0, maxAttachmentName),
			),
		),
	)
}

// exportedAddress returns the party's first address, the one gobl.ubl exports.
func exportedAddress(party *org.Party) *org.Address {
	if party == nil || len(party.Addresses) == 0 {
		return nil
	}

	return party.Addresses[0]
}

// addressesExported returns the first address of a slice, the only one exported.
func addressesExported(value any) *org.Address {
	addresses, ok := value.([]*org.Address)
	if !ok || len(addresses) == 0 {
		return nil
	}

	return addresses[0]
}

// addressInRomania reports an address ANAF applies its subdivision rules to.
func addressInRomania(address *org.Address) bool {
	return address != nil && address.Country == l10n.RO.ISO()
}

// addressHasStreet reports an address that fills BT-35, BT-50, BT-64 or BT-75, which gobl.ubl writes only from the street, a number alone leaving it empty (BR-RO-081).
func addressHasStreet(address *org.Address) bool {
	return address != nil && strings.TrimSpace(address.Street) != ""
}

// addressHasLocality reports an address that fills BT-37, BT-52, BT-66 or BT-77.
func addressHasLocality(address *org.Address) bool {
	return address != nil && strings.TrimSpace(address.Locality) != ""
}

// addressHasRegion reports an address that fills BT-39, BT-54, BT-68 or BT-79.
func addressHasRegion(address *org.Address) bool {
	return address != nil && strings.TrimSpace(address.Region) != ""
}

// partyAddressHasStreet applies BR-RO-080 and BR-RO-140. A missing party passes,
// leaving it to the rule that requires it.
func partyAddressHasStreet(value any) bool {
	party, ok := value.(*org.Party)
	if !ok || party == nil {
		return true
	}

	return addressHasStreet(exportedAddress(party))
}

// partyAddressHasLocality applies BR-RO-090 and BR-RO-150.
func partyAddressHasLocality(value any) bool {
	party, ok := value.(*org.Party)
	if !ok || party == nil {
		return true
	}

	return addressHasLocality(exportedAddress(party))
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

// attributeValue renders BT-161 the way gobl.ubl does, preferring the amount
// with its unit over the plain text.
func attributeValue(attribute *org.Attribute) string {
	if attribute == nil {
		return ""
	}

	if attribute.Amount != nil {
		value := attribute.Amount.String()
		if attribute.Unit != "" {
			value += attributeUnitSeparate + string(attribute.Unit)
		}
		return value
	}

	return attribute.Text
}

func attributeValueWithinLength(value any) bool {
	attribute, ok := value.(*org.Attribute)
	if !ok || attribute == nil {
		return true
	}

	return len([]rune(attributeValue(attribute))) <= maxAttributeValue
}

// RenderNote writes BT-22 and BT-127 the way gobl.ubl does, prefixing the text
// with the UNTDID 4451 subject when the note carries one.
func RenderNote(note *org.Note) string {
	if note == nil {
		return ""
	}

	if code := note.Ext.Get(untdid.ExtKeyTextSubject); code != cbc.CodeEmpty {
		return fmt.Sprintf("#%s#%s", code, note.Text)
	}

	return note.Text
}

func noteWithinLength(value any) bool {
	note, ok := value.(*org.Note)
	if !ok || note == nil {
		return true
	}

	return len([]rune(RenderNote(note))) <= maxNoteText
}

// accountingRefWithinLength covers BT-133, written from the note text alone.
func accountingRefWithinLength(value any) bool {
	note, ok := value.(*org.Note)
	if !ok || note == nil || note.Key != NoteKeyBuyerAccountingRef {
		return true
	}

	return len([]rune(note.Text)) <= maxNoteAccountingRef
}
