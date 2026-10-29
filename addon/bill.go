// Rules for the invoice as a whole, its delivery address, and its allowances and charges.
//
// Official sources:
//
//   - Order MF 4092/2022: the BR-RO table of national rules (art. 3 alin. (2))
//     https://static.anaf.ro/static/10/Anaf/legislatie/OMF_4092_2022.pdf
//   - EN 16931-1, BR-53: an invoice that names a VAT accounting currency also states its VAT
//     total in that currency
//   - OUG 120/2021 art. 10^1 alin. (3), as amended by OUG 138/2024: a consumer who gives no
//     tax code is invoiced under a code of thirteen zeros
//     https://static.anaf.ro/static/10/Anaf/legislatie/OUG_138_2024.pdf
//
// Each rule names the official rule it applies at the end of its message, such as
// (BR-RO-010). The addon README maps every BR-RO rule to the fault code GOBL reports.

package efactura

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

// TaxCurrency is the currency VAT is always stated in, BR-RO-030.
const TaxCurrency = currency.RON

// maxSubunits is the number of decimals BR-RO-Z2 allows on any amount.
const maxSubunits = 2

// Limits BR-RO-A020, A050, A500, L100 and L200 put on the document.
const (
	maxNotes           = 20
	maxPreceding       = 500
	maxAttachments     = 50
	maxInvoiceNumber   = 200
	maxPrecedingNumber = 200
	maxAccountingCost  = 100
	maxExemptionReason = 100
	maxReason          = 100
)

// digitRegexp is BR-RO-010: BT-1 has to carry at least one digit.
var digitRegexp = regexp.MustCompile(`[0-9]`)

// normalizeInvoice sets the rounding every amount is stated with and the
// identifier a consumer buyer is named by.
func normalizeInvoice(invoice *bill.Invoice) {
	normalizeRounding(invoice)
	normalizeCustomer(invoice)
}

// normalizeRounding makes every amount land on the two decimals BR-RO-Z2 allows.
// GOBL's default precision leaves totals such as 99.995 that ANAF rejects.
func normalizeRounding(invoice *bill.Invoice) {
	if invoice.Tax == nil {
		invoice.Tax = new(bill.Tax)
	}

	invoice.Tax.Rounding = tax.RoundingRuleCurrency
}

// normalizeCustomer gives a consumer buyer the BT-47 identifier BR-RO-120 needs:
// their CNP, or the thirteen zeros of OUG 120/2021 art. 10^1 alin. (3) when a buyer in Romania gave none.
func normalizeCustomer(invoice *bill.Invoice) {
	customer := invoice.Customer
	if customer == nil {
		return
	}

	// A tax number or an existing legal identity already fills BT-47
	if IsIdentified(customer) {
		return
	}

	// A consumer who gave their number: promote it out of BT-46 into BT-47
	if identity := FindCNP(customer); identity != nil {
		identity.Scope = org.IdentityScopeLegal
		return
	}

	// The thirteen zeros stand for a Romanian consumer, so a foreign buyer must state its own identifier
	if address := exportedAddress(customer); address != nil && !address.Country.Empty() && address.Country.Code() != l10n.RO {
		return
	}

	customer.Identities = org.AddIdentity(customer.Identities, &org.Identity{
		Scope: org.IdentityScopeLegal,
		Type:  IdentityTypeCNP,
		Code:  AnonymousCNP,
	})
}

func billInvoiceRules() *rules.Set {
	return rules.For(new(bill.Invoice),
		rules.Assert("01", "invoice number (with series when set) must include at least one digit (BR-RO-010)",
			is.Func("invoice number has a digit", invoiceNumberHasDigit),
		),
		rules.Assert("02", "invoice number (with series when set) must be no more than 200 characters long (BR-RO-L200)",
			is.Func("invoice number within length", invoiceNumberWithinLength),
		),
		rules.Assert("03", "invoice in a currency other than RON must provide an exchange rate to RON, so VAT can be stated in RON (BR-RO-030, BR-53)",
			currency.CanConvertTo(TaxCurrency),
		),
		rules.Assert("04", "invoice currency must state its amounts on no more than two decimals (BR-RO-Z2)",
			is.Func("currency within two subunits", currencyWithinSubunits),
		),
		rules.Field("notes",
			rules.Assert("05", "invoice notes must contain no more than 20 entries (BR-RO-A020)",
				is.Length(0, maxNotes),
			),
		),
		rules.Field("preceding",
			rules.Assert("06", "invoice preceding must contain no more than 500 entries (BR-RO-A500)",
				is.Length(0, maxPreceding),
			),
			rules.Assert("07", "preceding invoice number must be no more than 200 characters long (BR-RO-L200)",
				is.Func("preceding numbers within length", precedingNumbersWithinLength),
			),
		),
		rules.Field("attachments",
			rules.Assert("08", "invoice attachments must contain no more than 50 entries (BR-RO-A050)",
				is.Length(0, maxAttachments),
			),
		),
		rules.Field("ordering",
			rules.Field("cost",
				rules.Assert("09", "buyer accounting reference must be no more than 100 characters long (BR-RO-L100)",
					is.RuneLength(0, maxAccountingCost),
				),
			),
			rules.Field("seller",
				rules.Assert("16", "tax representative address line 1 is required (BR-RO-140)",
					is.Func("tax representative address has a street", partyAddressHasStreet),
				),
				rules.Assert("17", "tax representative city is required (BR-RO-150)",
					is.Func("tax representative address has a city", partyAddressHasLocality),
				),
			),
		),
		rules.Assert("10", "invoice supplier must have a VAT identifier or a tax registration identifier, or its tax representative a VAT identifier (BR-RO-065)",
			is.Func("seller is identified", sellerIdentified),
		),
		rules.Field("supplier",
			rules.Assert("11", "invoice supplier address line 1 is required (BR-RO-080)",
				is.Func("supplier address has a street", partyAddressHasStreet),
			),
			rules.Assert("12", "invoice supplier city is required (BR-RO-090)",
				is.Func("supplier address has a city", partyAddressHasLocality),
			),
		),
		rules.Field("customer",
			rules.Assert("13", "invoice customer must have a legal registration identifier or a VAT identifier (BR-RO-120)",
				is.Func("customer is identified", customerIdentified),
			),
			rules.Assert("14", "invoice customer address line 1 is required (BR-RO-080)",
				is.Func("customer address has a street", partyAddressHasStreet),
			),
			rules.Assert("15", "invoice customer city is required (BR-RO-090)",
				is.Func("customer address has a city", partyAddressHasLocality),
			),
		),
		rules.Field("tax",
			rules.Field("notes",
				rules.Assert("20", "VAT exemption reason must be no more than 100 characters long (BR-RO-L100)",
					is.Func("exemption reasons within length", exemptionReasonsWithinLength),
				),
			),
		),
		rules.When(
			bill.InvoiceTypeIn(bill.InvoiceTypeCreditNote),
			rules.Field("tax",
				rules.Field("ext",
					rules.Assert("21", "invoice tax ext untdid-document-type must be 381 for a credit note (BR-RO-020)",
						tax.ExtensionsHasCodes(untdid.ExtKeyDocumentType, CreditNoteTypes...),
					),
				),
			),
		),
		rules.When(
			invoiceIsNotCreditNote(),
			rules.Field("tax",
				rules.Field("ext",
					rules.Assert("22", "invoice tax ext untdid-document-type must be one of 380, 384, 389 or 751 (BR-RO-020)",
						tax.ExtensionsHasCodes(untdid.ExtKeyDocumentType, DocumentTypes...),
					),
				),
			),
		),
		rules.When(
			is.Func("invoice is issued in an enforcement procedure", invoiceIsEnforcement),
			rules.Assert("35", "an enforcement invoice must name the enforcement body as payee, with its name and its legal registration identifier (BR-RO-130)",
				is.Func("enforcement body identified", enforcementBodyIdentified),
			),
		),
	)
}

func billDeliveryRules() *rules.Set {
	return rules.For(new(bill.DeliveryDetails),
		rules.When(
			is.Func("delivery has an address", deliveryHasAddress),
			rules.Assert("01", "invoice delivery address line 1 is required when a delivery address is provided (BR-RO-180)",
				is.Func("delivery address has a street", deliveryAddressHasStreet),
			),
			rules.Assert("02", "invoice delivery city is required when a delivery address is provided (BR-RO-200)",
				is.Func("delivery address has a city", deliveryAddressHasLocality),
			),
			rules.Assert("03", "invoice delivery country subdivision is required when a delivery address is provided (BR-RO-210)",
				is.Func("delivery address has a region", deliveryAddressHasRegion),
			),
		),
	)
}

func billDiscountRules() *rules.Set {
	return rules.For(new(bill.Discount),
		rules.Field("reason",
			rules.Assert("01", "document level allowance reason must be no more than 100 characters long (BR-RO-L100)",
				is.RuneLength(0, maxReason),
			),
		),
	)
}

func billChargeRules() *rules.Set {
	return rules.For(new(bill.Charge),
		rules.Field("reason",
			rules.Assert("01", "document level charge reason must be no more than 100 characters long (BR-RO-L100)",
				is.RuneLength(0, maxReason),
			),
		),
	)
}

func billLineDiscountRules() *rules.Set {
	return rules.For(new(bill.LineDiscount),
		rules.Field("reason",
			rules.Assert("01", "invoice line allowance reason must be no more than 100 characters long (BR-RO-L100)",
				is.RuneLength(0, maxReason),
			),
		),
	)
}

func billLineChargeRules() *rules.Set {
	return rules.For(new(bill.LineCharge),
		rules.Field("reason",
			rules.Assert("01", "invoice line charge reason must be no more than 100 characters long (BR-RO-L100)",
				is.RuneLength(0, maxReason),
			),
		),
	)
}

// invoiceIsNotCreditNote covers every document ANAF files as an Invoice rather than a CreditNote.
func invoiceIsNotCreditNote() rules.Test {
	return is.Func("invoice is not a credit note", func(value any) bool {
		invoice, ok := value.(*bill.Invoice)

		return ok && invoice != nil && !invoice.Type.In(bill.InvoiceTypeCreditNote)
	})
}

// invoiceNumber renders BT-1 the way gobl.ubl does.
func invoiceNumber(invoice *bill.Invoice) string {
	if invoice.Series == "" {
		return invoice.Code.String()
	}

	return fmt.Sprintf("%s-%s", invoice.Series, invoice.Code)
}

func invoiceNumberHasDigit(value any) bool {
	invoice, ok := value.(*bill.Invoice)
	if !ok || invoice == nil {
		return true
	}

	return digitRegexp.MatchString(invoiceNumber(invoice))
}

func invoiceNumberWithinLength(value any) bool {
	invoice, ok := value.(*bill.Invoice)
	if !ok || invoice == nil {
		return true
	}

	return len([]rune(invoiceNumber(invoice))) <= maxInvoiceNumber
}

// currencyWithinSubunits keeps BR-RO-Z2 true: amounts are stated on the
// currency's own precision, so the currency itself may not exceed two decimals.
func currencyWithinSubunits(value any) bool {
	invoice, ok := value.(*bill.Invoice)
	if !ok || invoice == nil || invoice.Currency == currency.CodeEmpty {
		return true
	}

	def := invoice.Currency.Def()
	if def == nil {
		return true
	}

	return def.Subunits <= maxSubunits
}

// precedingNumbersWithinLength renders BT-25 the way gobl.ubl does.
func precedingNumbersWithinLength(value any) bool {
	refs, ok := value.([]*org.DocumentRef)
	if !ok {
		return true
	}

	for _, ref := range refs {
		if ref == nil {
			continue
		}
		if len([]rune(ref.Series.Join(ref.Code).String())) > maxPrecedingNumber {
			return false
		}
	}

	return true
}

// exemptionReasonsWithinLength covers BT-120, written from the tax notes.
func exemptionReasonsWithinLength(value any) bool {
	notes, ok := value.([]*tax.Note)
	if !ok {
		return true
	}

	for _, note := range notes {
		if note != nil && len([]rune(note.Text)) > maxExemptionReason {
			return false
		}
	}

	return true
}

// sellerIdentified applies BR-RO-065: the supplier's tax number (BT-31) or tax registration (BT-32), or the tax representative's tax number (BT-63).
func sellerIdentified(value any) bool {
	invoice, ok := value.(*bill.Invoice)
	if !ok || invoice == nil {
		return true
	}

	if HasTaxNumber(invoice.Supplier) || hasTaxRegistration(invoice.Supplier) {
		return true
	}

	return invoice.Ordering != nil && HasTaxNumber(invoice.Ordering.Seller)
}

func customerIdentified(value any) bool {
	party, ok := value.(*org.Party)

	return ok && IsIdentified(party)
}

func invoiceIsEnforcement(value any) bool {
	invoice, ok := value.(*bill.Invoice)

	return ok && IsEnforcement(invoice)
}

// enforcementBodyIdentified applies BR-RO-130: the payee names the enforcement body (BT-59) and gives its legal registration identifier (BT-61).
func enforcementBodyIdentified(value any) bool {
	invoice, ok := value.(*bill.Invoice)
	if !ok || invoice == nil {
		return true
	}

	if invoice.Payment == nil || invoice.Payment.Payee == nil {
		return false
	}
	payee := invoice.Payment.Payee

	return strings.TrimSpace(payee.Name) != "" && HasLegalIdentity(payee)
}

// deliveryAddress returns the receiver's first address, which is what gobl.ubl maps into the delivery location.
func deliveryAddress(value any) *org.Address {
	delivery, ok := value.(*bill.DeliveryDetails)
	if !ok || delivery == nil {
		return nil
	}

	return exportedAddress(delivery.Receiver)
}

func deliveryHasAddress(value any) bool {
	return deliveryAddress(value) != nil
}

func deliveryAddressHasStreet(value any) bool {
	return addressHasStreet(deliveryAddress(value))
}

func deliveryAddressHasLocality(value any) bool {
	return addressHasLocality(deliveryAddress(value))
}

func deliveryAddressHasRegion(value any) bool {
	return addressHasRegion(deliveryAddress(value))
}
