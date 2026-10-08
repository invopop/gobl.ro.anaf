package converter

import (
	"unicode"

	addon "github.com/invopop/gobl.ro.anaf/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
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

// delocalize restores what CIUS-RO and the parsers lose, mirroring localize
func delocalize(invoice *bill.Invoice, stated *stated) {
	delocalizeDocumentType(invoice, stated)
	delocalizeOutsideScope(invoice)
	delocalizeParties(invoice)
	delocalizePayment(invoice, stated)
	delocalizeAttachments(invoice, stated)
	delocalizeTaxCurrency(invoice, stated)
}

// delocalizeDocumentType restores the tags of BT-3: the parsers read 751 as other, and gobl.ubl drops the 389 tag when it keeps the stated totals
func delocalizeDocumentType(invoice *bill.Invoice, stated *stated) {
	switch stated.typeCode {
	case addon.DocumentTypeAccounting:
		invoice.Type = bill.InvoiceTypeStandard
		addTag(invoice, addon.TagAccounting)
	case addon.DocumentTypeSelfBilled:
		invoice.Type = bill.InvoiceTypeStandard
		addTag(invoice, tax.TagSelfBilled)
	}
}

// delocalizeOutsideScope undoes localizeOutsideScope, turning the CIFs moved to BT-32 and BT-47 back into tax identifiers
func delocalizeOutsideScope(invoice *bill.Invoice) {
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

// delocalizeParties restores the tax identifiers and CNPs of every party
func delocalizeParties(invoice *bill.Invoice) {
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

		// The parsers label every tax number VAT, which GOBL leaves implicit
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

// delocalizePayment restores the BT-81 code the parsers cannot map, and the SEPA creditor gobl.ubl leaves on the seller
func delocalizePayment(invoice *bill.Invoice, stated *stated) {
	if invoice.Payment == nil || invoice.Payment.Instructions == nil {
		return
	}
	instructions := invoice.Payment.Instructions

	// BT-81 with no GOBL means key
	if stated.paymentMeans != "" && instructions.Ext.Get(untdid.ExtKeyPaymentMeans) != stated.paymentMeans {
		instructions.Key = pay.MeansKeyOther
		instructions.Ext = instructions.Ext.Set(untdid.ExtKeyPaymentMeans, stated.paymentMeans)
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

// delocalizeAttachments restores the file names gobl.ubl drops, BT-125
func delocalizeAttachments(invoice *bill.Invoice, stated *stated) {
	for _, attachment := range invoice.Attachments {
		if attachment.Name == "" {
			attachment.Name = stated.fileNames[attachment.URL]
		}
	}
}

// delocalizeTaxCurrency undoes localizeTaxCurrency, deriving the exchange rate from the VAT in both currencies, BT-110 and BT-111
func delocalizeTaxCurrency(invoice *bill.Invoice, stated *stated) {
	if invoice.Currency == addon.TaxCurrency {
		return
	}

	// With no VAT there is no rate to recover
	vat, ok := stated.vat[invoice.Currency]
	if !ok || vat.IsZero() {
		return
	}

	vatRON, ok := stated.vat[addon.TaxCurrency]
	if !ok {
		return
	}

	invoice.ExchangeRates = []*currency.ExchangeRate{{
		From:   invoice.Currency,
		To:     addon.TaxCurrency,
		Amount: vatRON.Rescale(rateDecimals).Divide(vat),
	}}
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
