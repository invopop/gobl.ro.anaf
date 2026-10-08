package roefactura

import (
	addon "github.com/invopop/gobl.ro.anaf/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/currency"
)

// orderReferenceFiller is the placeholder gobl.ubl writes into BT-13 for Peppol
const orderReferenceFiller = "NA"

// localize applies the CIUS-RO details gobl.ubl cannot decide on its own
func localize(document *ubl.Invoice, invoice *bill.Invoice) error {
	// Drop the schema location, which points at local paths
	document.SchemaLocation = ""

	// Drop the purchase order placeholder
	dropOrderReferenceFiller(document, invoice)

	// Move the CIFs of a seller outside the VAT scope
	localizeOutsideScope(document, invoice)

	// State the VAT in RON
	return localizeTaxCurrency(document, invoice)
}

// taxSchemeRegistration is the BT-32 scheme of a tax registration that is not VAT
const taxSchemeRegistration = "TAX"

// localizeOutsideScope moves the CIFs out of the VAT fields for a seller outside the VAT scope (BR-O-02, BR-O-04)
func localizeOutsideScope(document *ubl.Invoice, invoice *bill.Invoice) {
	if !addon.IsOutsideScope(invoice) {
		return
	}

	// Pick the seller, preferring the ordering seller
	seller := invoice.Supplier
	if invoice.Ordering != nil && invoice.Ordering.Seller != nil {
		seller = invoice.Ordering.Seller
	}

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

// dropOrderReferenceFiller removes the BT-13 placeholder, keeping a real purchase order
func dropOrderReferenceFiller(document *ubl.Invoice, invoice *bill.Invoice) {
	if document.OrderReference == nil {
		return
	}

	if document.OrderReference.ID != orderReferenceFiller {
		return
	}

	if document.OrderReference.SalesOrderID != "" {
		return
	}

	if invoice.Ordering != nil && len(invoice.Ordering.Purchases) > 0 {
		return
	}

	document.OrderReference = nil
}

// ***** Delete once the regime is implemented
// localizeTaxCurrency writes BT-6 and BT-111, which gobl.ubl leaves out until GOBL has a Romanian regime
func localizeTaxCurrency(document *ubl.Invoice, invoice *bill.Invoice) error {
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
