package roefactura

import (
	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl.ro.anaf/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/tax"
)

// orderReferenceFiller is the placeholder gobl.ubl writes into BT-13 for Peppol
const orderReferenceFiller = "NA"

// taxSchemeRegistration is the BT-32 scheme of a tax registration that is not VAT
const taxSchemeRegistration = "TAX"

//
// Export funcs: run by FormatCIUSRO, in this order, after gobl.ubl writes the UBL document
//

// exportSchemaLocation drops the schema location, which gobl.ubl points at local paths ANAF cannot resolve
func exportSchemaLocation(_ *ubl.Format, _ *gobl.Envelope, doc ubl.Document) error {
	document, ok := doc.(*ubl.Invoice)
	if !ok {
		return nil
	}

	document.SchemaLocation = ""

	return nil
}

// exportOrderReference drops the BT-13 placeholder gobl.ubl writes for Peppol, keeping a real purchase order
func exportOrderReference(_ *ubl.Format, env *gobl.Envelope, doc ubl.Document) error {
	document, invoice, ok := exported(env, doc)
	if !ok || document.OrderReference == nil {
		return nil
	}

	if document.OrderReference.ID != orderReferenceFiller || document.OrderReference.SalesOrderID != "" {
		return nil
	}

	if invoice.Ordering != nil && len(invoice.Ordering.Purchases) > 0 {
		return nil
	}

	document.OrderReference = nil

	return nil
}

// exportOutsideScope moves the CIFs out of the VAT fields for a seller not registered for VAT, category O (BR-O-02, BR-O-04)
func exportOutsideScope(_ *ubl.Format, env *gobl.Envelope, doc ubl.Document) error {
	document, invoice, ok := exported(env, doc)
	if !ok || !addon.IsOutsideScope(invoice) {
		return nil
	}

	// The seller is the supplier, the ordering seller being its tax representative (BG-11)
	seller := invoice.Supplier

	// Seller CIF from BT-31 to BT-32 and BT-30
	if party := document.AccountingSupplierParty.Party; party != nil && addon.HasTaxNumber(seller) {
		party.PartyTaxScheme = []ubl.PartyTaxScheme{{
			CompanyID: &ubl.IDType{Value: addon.CIF(seller)},
			TaxScheme: &ubl.TaxScheme{ID: ubl.IDType{Value: taxSchemeRegistration}},
		}}
		setLegalCompanyID(party, addon.CIF(seller))
	}

	// Buyer CIF from BT-48 to BT-47
	if party := document.AccountingCustomerParty.Party; party != nil && addon.HasTaxNumber(invoice.Customer) {
		party.PartyTaxScheme = dropVATSchemes(party.PartyTaxScheme)
		setLegalCompanyID(party, addon.CIF(invoice.Customer))
	}

	return nil
}

// exportSelfBillingNote writes the autofactura mention, which the addon's scenario keeps in the tax notes gobl.ubl does not print, into BT-22 (BR-RO-060), unless a note already carries it
func exportSelfBillingNote(_ *ubl.Format, env *gobl.Envelope, doc ubl.Document) error {
	document, invoice, ok := exported(env, doc)
	if !ok || invoice.Tax == nil || hasSelfBillingMention(document.Note) {
		return nil
	}

	for _, note := range invoice.Tax.Notes {
		if note != nil && note.Key == tax.TagSelfBilled {
			document.Note = append(document.Note, note.Text)
		}
	}

	return nil
}

// ***** Delete once the regime is implemented
// exportTaxCurrency writes BT-6 and BT-111, which gobl.ubl leaves out until GOBL has a Romanian regime (BR-RO-030)
func exportTaxCurrency(_ *ubl.Format, env *gobl.Envelope, doc ubl.Document) error {
	document, invoice, ok := exported(env, doc)
	if !ok {
		return nil
	}

	// No tax currency for an invoice in RON
	if document.DocumentCurrencyCode == addon.TaxCurrency.String() {
		document.TaxCurrencyCode = ""
		return nil
	}

	// The rate and totals the RON amount is built from
	rate := currency.MatchExchangeRate(invoice.ExchangeRates, invoice.Currency, addon.TaxCurrency)
	if rate == nil {
		return ErrNotCompliant.WithMsg("invoice in %s states no exchange rate to %s, so VAT cannot be given in the accounting currency (BR-RO-030)", invoice.Currency, addon.TaxCurrency)
	}

	if invoice.Totals == nil {
		return ErrConversion.WithMsg("invoice carries no totals, so the VAT amount in %s cannot be stated", addon.TaxCurrency)
	}

	// BT-6
	document.TaxCurrencyCode = addon.TaxCurrency.String()

	// BT-111, rescaled to two decimals (BR-RO-Z2)
	taxCurrency := addon.TaxCurrency.String()
	document.TaxTotal = append(document.TaxTotal, ubl.TaxTotal{
		TaxAmount: ubl.Amount{
			Value:      addon.TaxCurrency.Def().Rescale(rate.Convert(invoice.Totals.Tax)).String(),
			CurrencyID: &taxCurrency,
		},
	})

	return nil
}

//
// Helpers of the export funcs
//

// exported returns the UBL document and the invoice it was written from, or false for any other document
func exported(env *gobl.Envelope, doc ubl.Document) (*ubl.Invoice, *bill.Invoice, bool) {
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

// hasSelfBillingMention reports a BT-22 note that already carries the autofactura mention, with or without diacritics
func hasSelfBillingMention(notes []string) bool {
	for _, note := range notes {
		if addon.MentionsSelfBilling(note) {
			return true
		}
	}

	return false
}

// setLegalCompanyID fills BT-30 or BT-47, keeping one the party already states
func setLegalCompanyID(party *ubl.Party, code string) {
	if party.PartyLegalEntity == nil {
		party.PartyLegalEntity = new(ubl.PartyLegalEntity)
	}

	if party.PartyLegalEntity.CompanyID == nil {
		party.PartyLegalEntity.CompanyID = &ubl.IDType{Value: code}
	}
}

// dropVATSchemes keeps the tax schemes that are not VAT
func dropVATSchemes(schemes []ubl.PartyTaxScheme) []ubl.PartyTaxScheme {
	kept := make([]ubl.PartyTaxScheme, 0, len(schemes))
	for _, scheme := range schemes {
		if scheme.TaxScheme != nil && scheme.TaxScheme.ID.Value == ubl.TaxSchemeVAT {
			continue
		}
		kept = append(kept, scheme)
	}

	if len(kept) == 0 {
		return nil
	}

	return kept
}
