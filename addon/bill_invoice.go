package efactura

import (
	"fmt"
	"regexp"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
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

// TaxCategoryIntraCommunity is the UNTDID 5305 code of an intra-community supply.
const TaxCategoryIntraCommunity cbc.Code = "K"

// Limits the RO_CIUS puts on repeated groups and document identifiers.
const (
	maxNotes           = 20
	maxPreceding       = 500
	maxAttachments     = 50
	maxInvoiceNumber   = 200
	maxPrecedingNumber = 200
	maxAccountingCost  = 100
	maxExemptionReason = 100
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

// normalizeCustomer gives a Romanian consumer buyer the BT-47 identifier
// BR-RO-120 needs: their CNP, or thirteen zeros when they gave none. A foreign
// buyer is left as written, having no CNP to stand in for.
func normalizeCustomer(invoice *bill.Invoice) {
	customer := invoice.Customer
	if customer == nil {
		return
	}

	// A tax number or an existing legal identity already fills BT-47.
	if IsIdentified(customer) {
		return
	}

	// A consumer who gave their number: promote it out of BT-46 into BT-47.
	if identity := FindCNP(customer); identity != nil {
		identity.Scope = org.IdentityScopeLegal
		return
	}

	if !isRomanianConsumer(customer) {
		return
	}

	customer.Identities = org.AddIdentity(customer.Identities, &org.Identity{
		Scope: org.IdentityScopeLegal,
		Type:  IdentityTypeCNP,
		Code:  AnonymousCNP,
	})
}

// isRomanianConsumer reports a natural person the placeholder CNP belongs to. A
// missing country reads as Romanian, since EN 16931 rejects it first anyway.
func isRomanianConsumer(party *org.Party) bool {
	address := exportedAddress(party)
	if address == nil {
		return true
	}

	return address.Country == "" || address.Country == l10n.RO.ISO()
}

func billInvoiceRules() *rules.Set {
	return rules.For(new(bill.Invoice),
		rules.Assert("01", "invoice number (BT-1, with series when set) must include at least one digit (BR-RO-010)",
			is.Func("invoice number has a digit", invoiceNumberHasDigit),
		),
		rules.Assert("02", "invoice number (BT-1, with series when set) must be no more than 200 characters long (BR-RO-L200)",
			is.Func("invoice number within length", invoiceNumberWithinLength),
		),
		rules.Assert("03", "invoice must be in RON or provide an exchange rate to RON, so VAT can be stated in the accounting currency (BR-RO-030)",
			currency.CanConvertTo(TaxCurrency),
		),
		rules.Assert("04", "invoice currency must state its amounts on no more than two decimals (BR-RO-Z2)",
			is.Func("currency within two subunits", currencyWithinSubunits),
		),
		rules.Field("notes",
			rules.Assert("05", "invoice notes (BG-1) must contain no more than 20 entries (BR-RO-A020)",
				is.Length(0, maxNotes),
			),
		),
		rules.Field("preceding",
			rules.Assert("06", "invoice preceding (BG-3) must contain no more than 500 entries (BR-RO-A500)",
				is.Length(0, maxPreceding),
			),
			rules.Assert("07", "preceding invoice number (BT-25) must be no more than 200 characters long (BR-RO-L200)",
				is.Func("preceding numbers within length", precedingNumbersWithinLength),
			),
		),
		rules.Field("attachments",
			rules.Assert("08", "invoice attachments (BG-24) must contain no more than 50 entries (BR-RO-A050)",
				is.Length(0, maxAttachments),
			),
		),
		rules.Field("ordering",
			rules.Field("cost",
				rules.Assert("09", "buyer accounting reference (BT-19) must be no more than 100 characters long (BR-RO-L100)",
					is.RuneLength(0, maxAccountingCost),
				),
			),
			rules.Field("seller",
				rules.Assert("16", "invoice ordering seller address line 1 (BT-35) is required (BR-RO-080)",
					is.Func("ordering seller address has a street", partyAddressHasStreet),
				),
				rules.Assert("17", "invoice ordering seller city (BT-37) is required (BR-RO-090)",
					is.Func("ordering seller address has a city", partyAddressHasLocality),
				),
			),
		),
		rules.Field("supplier",
			rules.Assert("10", "invoice supplier must have a tax identifier (BT-31, BT-32) (BR-RO-065)",
				is.Func("supplier has a tax number", supplierIdentified),
			),
			rules.Assert("11", "invoice supplier address line 1 (BT-35) is required (BR-RO-080)",
				is.Func("supplier address has a street", partyAddressHasStreet),
			),
			rules.Assert("12", "invoice supplier city (BT-37) is required (BR-RO-090)",
				is.Func("supplier address has a city", partyAddressHasLocality),
			),
		),
		rules.Field("customer",
			rules.Assert("13", "invoice customer must have a VAT identifier (BT-48) or a legal registration identifier (BT-47) (BR-RO-120)",
				is.Func("customer is identified", customerIdentified),
			),
			rules.Assert("14", "invoice customer address line 1 (BT-50) is required (BR-RO-080)",
				is.Func("customer address has a street", partyAddressHasStreet),
			),
			rules.Assert("15", "invoice customer city (BT-52) is required (BR-RO-090)",
				is.Func("customer address has a city", partyAddressHasLocality),
			),
		),
		rules.Field("tax",
			rules.Assert("18", "invoice tax is required", is.Present),
			rules.Field("ext",
				rules.Assert("19", "invoice tax ext untdid-document-type is required",
					tax.ExtensionsRequire(untdid.ExtKeyDocumentType),
				),
			),
			rules.Field("notes",
				rules.Assert("20", "VAT exemption reason (BT-120) must be no more than 100 characters long (BR-RO-L100)",
					is.Func("exemption reasons within length", exemptionReasonsWithinLength),
				),
			),
		),
		rules.When(
			bill.InvoiceTypeIn(bill.InvoiceTypeCreditNote),
			rules.Field("tax",
				rules.Field("ext",
					rules.Assert("21", "invoice tax ext untdid-document-type must be 381 for a credit note (BR-RO-020_2)",
						tax.ExtensionsHasCodes(untdid.ExtKeyDocumentType, CreditNoteTypes...),
					),
				),
			),
		),
		rules.When(
			invoiceIsNotCreditNote(),
			rules.Field("tax",
				rules.Field("ext",
					rules.Assert("22", "invoice tax ext untdid-document-type must be one of 380, 384, 389 or 751 (BR-RO-020_1)",
						tax.ExtensionsHasCodes(untdid.ExtKeyDocumentType, DocumentTypes...),
					),
				),
			),
		),
		rules.When(
			is.Func("invoice is self-billed", invoiceIsSelfBilled),
			rules.Field("customer",
				rules.Assert("25", "a self-billed invoice must give the buyer a tax identifier (BT-48), because the buyer issues it and ANAF files it under the buyer's CIF",
					is.Func("buyer has a tax number", supplierIdentified),
				),
			),
		),
		rules.When(
			is.Func("invoice is an intra-community supply", invoiceIsIntraCommunity),
			rules.Assert("23", "an intra-community supply must state a delivery date (BT-72) or an invoicing period (BG-14) (BR-IC-11)",
				is.Func("supply has a delivery date or period", invoiceHasDeliveryDateOrPeriod),
			),
			rules.Assert("24", "an intra-community supply must state the delivery country (BT-80) (BR-IC-12)",
				is.Func("supply has a delivery country", invoiceHasDeliveryCountry),
			),
		),
	)
}

// invoiceIsNotCreditNote covers everything BR-RO-020_1 applies to, every
// document ANAF files as an Invoice rather than a CreditNote.
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

func supplierIdentified(value any) bool {
	party, ok := value.(*org.Party)

	return ok && HasTaxNumber(party)
}

func customerIdentified(value any) bool {
	party, ok := value.(*org.Party)

	return ok && IsIdentified(party)
}

// invoiceIsSelfBilled reports a document the buyer issues in the supplier's
// name, the one case where the buyer's tax number is mandatory.
func invoiceIsSelfBilled(value any) bool {
	invoice, ok := value.(*bill.Invoice)

	return ok && IsSelfBilled(invoice)
}

// invoiceIsIntraCommunity reports the intra-community VAT category anywhere in
// the breakdown: on a line, or on a document level allowance or charge.
func invoiceIsIntraCommunity(value any) bool {
	invoice, ok := value.(*bill.Invoice)
	if !ok || invoice == nil {
		return false
	}

	for _, line := range invoice.Lines {
		if line != nil && setIsIntraCommunity(line.Taxes) {
			return true
		}
	}

	for _, discount := range invoice.Discounts {
		if discount != nil && setIsIntraCommunity(discount.Taxes) {
			return true
		}
	}

	for _, charge := range invoice.Charges {
		if charge != nil && setIsIntraCommunity(charge.Taxes) {
			return true
		}
	}

	return false
}

func setIsIntraCommunity(set tax.Set) bool {
	for _, combo := range set {
		if combo == nil {
			continue
		}
		if combo.Key == tax.KeyIntraCommunity || combo.Rate == tax.KeyIntraCommunity {
			return true
		}
		if combo.Ext.Get(untdid.ExtKeyTaxCategory) == TaxCategoryIntraCommunity {
			return true
		}
	}

	return false
}

func invoiceHasDeliveryDateOrPeriod(value any) bool {
	invoice, ok := value.(*bill.Invoice)
	if !ok || invoice == nil {
		return false
	}

	if invoice.Delivery != nil && invoice.Delivery.Date != nil {
		return true
	}

	return invoice.Ordering != nil && invoice.Ordering.Period != nil
}

func invoiceHasDeliveryCountry(value any) bool {
	invoice, ok := value.(*bill.Invoice)
	if !ok || invoice == nil || invoice.Delivery == nil {
		return false
	}

	address := exportedAddress(invoice.Delivery.Receiver)

	return address != nil && address.Country != ""
}
