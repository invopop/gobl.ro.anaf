package roefactura

import (
	"unicode"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	addon "github.com/invopop/gobl.ro.anaf/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

// cnpLength is the number of digits of a Romanian personal numeric code
const cnpLength = 13

// rateDecimals is the precision BNR publishes its exchange rates with
const rateDecimals = 4

// schemeSEPA marks the SEPA creditor identifier, BT-90
const schemeSEPA cbc.Code = "SEPA"

// paymentMeansAny is the BT-81 code the EN 16931 addon gives the payment means key any
const paymentMeansAny cbc.Code = "1"

//
// UBL import funcs: run by FormatCIUSRO, in this order, after gobl.ubl reads the document and before it is calculated
//

// importDocumentType restores the tags of BT-3: gobl.ubl reads 751 as another type, and drops the 389 tag when it keeps the stated totals
func importDocumentType(_ *ubl.Format, doc ubl.Document, env *gobl.Envelope) error {
	document, invoice, ok := importedUBL(doc, env)
	if !ok {
		return nil
	}

	code := cbc.CodeEmpty
	if document.InvoiceTypeCode != nil {
		code = cbc.Code(document.InvoiceTypeCode.Value)
	}
	if document.CreditNoteTypeCode != nil {
		code = cbc.Code(document.CreditNoteTypeCode.Value)
	}

	restoreDocumentType(invoice, code)

	return nil
}

// importOutsideScope turns the CIFs exportOutsideScope moved to BT-32 and BT-47 back into tax identifiers
func importOutsideScope(_ *ubl.Format, doc ubl.Document, env *gobl.Envelope) error {
	if _, invoice, ok := importedUBL(doc, env); ok {
		restoreOutsideScope(invoice)
	}

	return nil
}

// importParties restores the tax identifiers and CNPs of every party
func importParties(_ *ubl.Format, doc ubl.Document, env *gobl.Envelope) error {
	if _, invoice, ok := importedUBL(doc, env); ok {
		restoreParties(invoice)
	}

	return nil
}

// importPaymentMeans restores the BT-81 code gobl.ubl cannot map, and the SEPA creditor it leaves on the seller
func importPaymentMeans(_ *ubl.Format, doc ubl.Document, env *gobl.Envelope) error {
	document, invoice, ok := importedUBL(doc, env)
	if !ok || len(document.PaymentMeans) == 0 {
		return nil
	}

	restorePaymentMeans(invoice, cbc.Code(document.PaymentMeans[0].PaymentMeansCode.Value))

	return nil
}

// importAttachmentNames restores the file names gobl.ubl drops, BT-125
func importAttachmentNames(_ *ubl.Format, doc ubl.Document, env *gobl.Envelope) error {
	document, invoice, ok := importedUBL(doc, env)
	if !ok {
		return nil
	}

	names := make(map[string]string)
	for _, reference := range document.AdditionalDocumentReference {
		if reference.Attachment != nil && reference.Attachment.ExternalReference != nil {
			names[reference.Attachment.ExternalReference.URI] = reference.Attachment.ExternalReference.FileName
		}
	}

	for _, attachment := range invoice.Attachments {
		if attachment.Name == "" {
			attachment.Name = names[attachment.URL]
		}
	}

	return nil
}

// importTaxCurrency derives the exchange rate to RON from the VAT in both currencies, BT-110 and BT-111, which exportTaxCurrency writes
func importTaxCurrency(_ *ubl.Format, doc ubl.Document, env *gobl.Envelope) error {
	document, invoice, ok := importedUBL(doc, env)
	if !ok {
		return nil
	}

	vat := make(map[currency.Code]num.Amount)
	for _, total := range document.TaxTotal {
		amount, err := num.AmountFromString(total.TaxAmount.Value)
		if err == nil && total.TaxAmount.CurrencyID != nil {
			vat[currency.Code(*total.TaxAmount.CurrencyID)] = amount
		}
	}

	restoreTaxCurrency(invoice, vat)

	return nil
}

// importSelfBillingNote drops the autofactura mention of a buyer-issued 389, which the addon's scenario adds back as a tax note
func importSelfBillingNote(_ *ubl.Format, doc ubl.Document, env *gobl.Envelope) error {
	if _, invoice, ok := importedUBL(doc, env); ok {
		restoreSelfBillingNote(invoice)
	}

	return nil
}

//
// CII import funcs: run by FormatCIUSROCII, in this order, after gobl.cii reads the document and before it is calculated
//

// importCIIDocumentType restores the tags of BT-3, as importDocumentType does for UBL
func importCIIDocumentType(_ *cii.Format, doc cii.Document, env *gobl.Envelope) error {
	document, invoice, ok := importedCII(doc, env)
	if !ok || document.ExchangedDocument == nil {
		return nil
	}

	restoreDocumentType(invoice, cbc.Code(document.ExchangedDocument.TypeCode))

	return nil
}

// importCIIOutsideScope turns the CIFs in BT-32 and BT-47 back into tax identifiers, as importOutsideScope does for UBL
func importCIIOutsideScope(_ *cii.Format, doc cii.Document, env *gobl.Envelope) error {
	if _, invoice, ok := importedCII(doc, env); ok {
		restoreOutsideScope(invoice)
	}

	return nil
}

// importCIIParties restores the tax identifiers and CNPs of every party, as importParties does for UBL
func importCIIParties(_ *cii.Format, doc cii.Document, env *gobl.Envelope) error {
	if _, invoice, ok := importedCII(doc, env); ok {
		restoreParties(invoice)
	}

	return nil
}

// importCIIPaymentMeans restores the BT-81 code gobl.cii cannot map, as importPaymentMeans does for UBL
func importCIIPaymentMeans(_ *cii.Format, doc cii.Document, env *gobl.Envelope) error {
	document, invoice, ok := importedCII(doc, env)
	if !ok || document.Transaction == nil || document.Transaction.Settlement == nil || len(document.Transaction.Settlement.PaymentMeans) == 0 {
		return nil
	}

	restorePaymentMeans(invoice, cbc.Code(document.Transaction.Settlement.PaymentMeans[0].TypeCode))

	return nil
}

// importCIITaxCurrency sets the exchange rate to RON from both VAT totals, BT-110 and BT-111, as importTaxCurrency does for UBL
func importCIITaxCurrency(_ *cii.Format, doc cii.Document, env *gobl.Envelope) error {
	document, invoice, ok := importedCII(doc, env)
	if !ok || document.Transaction == nil || document.Transaction.Settlement == nil || document.Transaction.Settlement.Summary == nil {
		return nil
	}

	vat := make(map[currency.Code]num.Amount)
	for _, total := range document.Transaction.Settlement.Summary.TaxTotalAmount {
		amount, err := num.AmountFromString(total.Amount)
		if err == nil && total.Currency != "" {
			vat[currency.Code(total.Currency)] = amount
		}
	}

	restoreTaxCurrency(invoice, vat)

	return nil
}

// importCIISelfBillingNote drops the autofactura mention of a buyer-issued 389, as importSelfBillingNote does for UBL
func importCIISelfBillingNote(_ *cii.Format, doc cii.Document, env *gobl.Envelope) error {
	if _, invoice, ok := importedCII(doc, env); ok {
		restoreSelfBillingNote(invoice)
	}

	return nil
}

//
// Helpers of the import funcs, shared by UBL and CII
//

// importedUBL returns the UBL document and the invoice read from it, or false for any other document
func importedUBL(doc ubl.Document, env *gobl.Envelope) (*ubl.Invoice, *bill.Invoice, bool) {
	document, ok := doc.(*ubl.Invoice)
	if !ok {
		return nil, nil, false
	}

	invoice, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, nil, false
	}

	return document, invoice, true
}

// importedCII returns the CII document and the invoice read from it, or false for any other document
func importedCII(doc cii.Document, env *gobl.Envelope) (*cii.Invoice, *bill.Invoice, bool) {
	document, ok := doc.(*cii.Invoice)
	if !ok {
		return nil, nil, false
	}

	invoice, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, nil, false
	}

	return document, invoice, true
}

// restoreDocumentType gives a 751 the accounting tag and a 389 the self-billed tag, both on a standard invoice
func restoreDocumentType(invoice *bill.Invoice, code cbc.Code) {
	switch code {
	case addon.DocumentTypeAccounting:
		invoice.Type = bill.InvoiceTypeStandard
		addTag(invoice, addon.TagAccounting)
	case addon.DocumentTypeSelfBilled:
		invoice.Type = bill.InvoiceTypeStandard
		addTag(invoice, tax.TagSelfBilled)
	}
}

// restoreOutsideScope turns the CIFs of an invoice in category O, kept in BT-32 and BT-47, back into tax identifiers
func restoreOutsideScope(invoice *bill.Invoice) {
	if !addon.IsOutsideScope(invoice) {
		return
	}

	// BT-32 back to the seller's tax identifier
	if supplier := invoice.Supplier; supplier != nil && supplier.TaxID == nil {
		if identity := findIdentity(supplier, org.IdentityScopeTax); identity != nil {
			supplier.TaxID = &tax.Identity{Country: l10n.TaxCountryCode(identity.Country), Code: identity.Code}
			supplier.Identities = dropIdentities(supplier.Identities, identity.Code)
		}
	}

	// BT-47 back to the buyer's tax identifier, when it is a CIF and not a CNP
	if customer := invoice.Customer; customer != nil && customer.TaxID == nil {
		if identity := findIdentity(customer, org.IdentityScopeLegal); identity != nil && isDigits(identity.Code) && !isCNP(identity.Code) {
			customer.TaxID = &tax.Identity{Country: partyCountry(customer), Code: identity.Code}
			customer.Identities = dropIdentities(customer.Identities, identity.Code)
		}
	}
}

// restoreParties drops the VAT label the readers add, takes the country from the tax number prefix, and types a 13-digit legal identifier as a CNP
func restoreParties(invoice *bill.Invoice) {
	parties := []*org.Party{invoice.Supplier, invoice.Customer}
	if invoice.Ordering != nil {
		parties = append(parties, invoice.Ordering.Seller)
	}
	if invoice.Payment != nil {
		parties = append(parties, invoice.Payment.Payee)
	}

	for _, party := range parties {
		if party == nil {
			continue
		}

		// The readers label every tax number VAT, which GOBL leaves implicit
		if party.TaxID != nil && party.TaxID.Scheme == ubl.TaxSchemeVAT {
			party.TaxID.Scheme = ""
		}

		// The prefix of a tax number names its country, whatever the address says
		if party.TaxID != nil {
			if country := taxPrefix(party.TaxID.Code); country != "" {
				party.TaxID.Country = country
			}
		}

		// A bare 13-digit legal identifier is a CNP
		for _, identity := range party.Identities {
			if identity.Scope == org.IdentityScopeLegal && identity.Type == "" && identity.Ext.IsZero() && isCNP(identity.Code) {
				identity.Type = addon.IdentityTypeCNP
			}
		}
	}
}

// restorePaymentMeans puts back a BT-81 code with no GOBL means key, and moves a SEPA creditor left on the seller back to the direct debit, BT-90
func restorePaymentMeans(invoice *bill.Invoice, code cbc.Code) {
	if invoice.Payment == nil || invoice.Payment.Instructions == nil {
		return
	}
	instructions := invoice.Payment.Instructions

	// BT-81 with no GOBL means key: the readers give it the key any, which calculation would turn into code 1
	unmapped := instructions.Key == pay.MeansKeyAny && code != paymentMeansAny
	if code != "" && (unmapped || instructions.Ext.Get(untdid.ExtKeyPaymentMeans) != code) {
		instructions.Key = pay.MeansKeyOther
		instructions.Ext = instructions.Ext.Set(untdid.ExtKeyPaymentMeans, code)
	}

	// BT-90 back from the seller to the direct debit
	if instructions.DirectDebit != nil && instructions.DirectDebit.Creditor == "" && invoice.Supplier != nil {
		for _, identity := range invoice.Supplier.Identities {
			if identity.Ext.Get(iso.ExtKeySchemeID) == schemeSEPA {
				instructions.DirectDebit.Creditor = identity.Code.String()
				invoice.Supplier.Identities = dropIdentities(invoice.Supplier.Identities, identity.Code)
				break
			}
		}
	}
}

// restoreTaxCurrency sets the exchange rate to RON as the VAT in RON over the VAT in the invoice currency, to the 4 decimals BNR publishes
func restoreTaxCurrency(invoice *bill.Invoice, vat map[currency.Code]num.Amount) {
	if invoice.Currency == addon.TaxCurrency {
		return
	}

	// With no VAT there is no rate to recover
	amount, ok := vat[invoice.Currency]
	if !ok || amount.IsZero() {
		return
	}

	amountRON, ok := vat[addon.TaxCurrency]
	if !ok {
		return
	}

	invoice.ExchangeRates = []*currency.ExchangeRate{{
		From:   invoice.Currency,
		To:     addon.TaxCurrency,
		Amount: amountRON.Rescale(rateDecimals).Divide(amount),
	}}
}

// restoreSelfBillingNote drops the autofactura mention from the notes of a buyer-issued invoice, since the addon's scenario adds it back as a tax note
func restoreSelfBillingNote(invoice *bill.Invoice) {
	if !addon.IsBuyerIssued(invoice) {
		return
	}

	kept := make([]*org.Note, 0, len(invoice.Notes))
	for _, note := range invoice.Notes {
		if note != nil && note.Text == addon.SelfBillingNote {
			continue
		}
		kept = append(kept, note)
	}

	if len(kept) == 0 {
		kept = nil
	}
	invoice.Notes = kept
}

// addTag adds a tag the invoice does not carry yet
func addTag(invoice *bill.Invoice, tag cbc.Key) {
	if !invoice.HasTags(tag) {
		invoice.SetTags(append(invoice.GetTags(), tag)...)
	}
}

// findIdentity returns the first untyped identity of the scope
func findIdentity(party *org.Party, scope cbc.Key) *org.Identity {
	for _, identity := range party.Identities {
		if identity.Scope == scope && identity.Type == "" && identity.Ext.IsZero() {
			return identity
		}
	}

	return nil
}

// dropIdentities removes the identities with the code
func dropIdentities(identities []*org.Identity, code cbc.Code) []*org.Identity {
	kept := make([]*org.Identity, 0, len(identities))
	for _, identity := range identities {
		if identity.Code != code {
			kept = append(kept, identity)
		}
	}

	if len(kept) == 0 {
		return nil
	}

	return kept
}

// partyCountry returns the country of the party's address
func partyCountry(party *org.Party) l10n.TaxCountryCode {
	if len(party.Addresses) == 0 {
		return ""
	}

	return l10n.TaxCountryCode(party.Addresses[0].Country)
}

// taxPrefix returns the country a tax number starts with, if any
func taxPrefix(code cbc.Code) l10n.TaxCountryCode {
	runes := []rune(code.String())
	if len(runes) < 3 || !unicode.IsLetter(runes[0]) || !unicode.IsLetter(runes[1]) {
		return ""
	}

	country := l10n.TaxCountryCode(string(runes[:2]))
	if country.Validate() != nil {
		return ""
	}

	return country
}

// isCNP reports whether the code has the shape of a CNP
func isCNP(code cbc.Code) bool {
	return len(code) == cnpLength && isDigits(code)
}

// isDigits reports whether the code is made of digits only
func isDigits(code cbc.Code) bool {
	if code == "" {
		return false
	}

	for _, digit := range code {
		if !unicode.IsDigit(digit) {
			return false
		}
	}

	return true
}
