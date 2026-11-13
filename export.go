package roefactura

import (
	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl.ro.anaf/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
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

// exportBindingParties keeps every party to its EN 16931 business terms, ANAF refusing the rest as UBL-CR warnings
func exportBindingParties(_ *ubl.Format, _ *gobl.Envelope, doc ubl.Document) error {
	document, ok := doc.(*ubl.Invoice)
	if !ok {
		return nil
	}

	// Seller: no service provider, the ordering issuer (UBL-CR-195)
	if party := document.AccountingSupplierParty.Party; party != nil {
		party.ServiceProviderParty = nil
		trimAddress(party.PostalAddress)
	}

	// Buyer: no BT-33, which EN 16931 gives the seller only (UBL-CR-244)
	if party := document.AccountingCustomerParty.Party; party != nil {
		trimAddress(party.PostalAddress)
		if party.PartyLegalEntity != nil {
			party.PartyLegalEntity.CompanyLegalForm = nil
		}
	}

	// Payee: BT-59, BT-60 and BT-61 only, its tax id moved to BT-61 when that is free (UBL-CR-269, 274, 279, 288)
	if party := document.PayeeParty; party != nil {
		var legal *ubl.IDType
		if party.PartyLegalEntity != nil {
			legal = party.PartyLegalEntity.CompanyID
		}
		if legal == nil && len(party.PartyTaxScheme) > 0 {
			legal = party.PartyTaxScheme[0].CompanyID
		}

		document.PayeeParty = &ubl.Party{
			PartyIdentification: party.PartyIdentification,
			PartyName:           party.PartyName,
		}
		if legal != nil {
			document.PayeeParty.PartyLegalEntity = &ubl.PartyLegalEntity{CompanyID: legal}
		}
	}

	// Tax representative: BT-62, BG-12 and BT-63 only (UBL-CR-300, 302, 335, 336)
	if party := document.TaxRepresentativeParty; party != nil {
		trimAddress(party.PostalAddress)
		document.TaxRepresentativeParty = &ubl.Party{
			PartyName:      party.PartyName,
			PostalAddress:  party.PostalAddress,
			PartyTaxScheme: party.PartyTaxScheme,
		}
	}

	// Deliver-to: BT-70 as the only party detail (UBL-CR-390 to 403)
	for _, delivery := range document.Delivery {
		if delivery == nil || delivery.DeliveryParty == nil {
			continue
		}
		name := delivery.DeliveryParty.PartyName
		delivery.DeliveryParty = nil
		if name != nil {
			delivery.DeliveryParty = &ubl.Party{PartyName: name}
		}
	}

	return nil
}

// exportBindingReferences keeps document references and attachments to their EN 16931 business terms
func exportBindingReferences(_ *ubl.Format, _ *gobl.Envelope, doc ubl.Document) error {
	document, ok := doc.(*ubl.Invoice)
	if !ok {
		return nil
	}

	// BT-12, BT-15, BT-16, BT-17 and BT-25 carry no type (UBL-CR-026, 099 and alike)
	for _, references := range [][]ubl.Reference{
		document.ContractDocumentReference,
		document.DespatchDocumentReference,
		document.ReceiptDocumentReference,
		document.OriginatorDocumentReference,
	} {
		for i := range references {
			trimReference(&references[i])
		}
	}
	for _, reference := range document.BillingReference {
		if reference != nil && reference.InvoiceDocumentReference != nil {
			trimReference(reference.InvoiceDocumentReference)
		}
	}

	// BG-24 has no uuid, hash, MIME type or file name next to an external location (UBL-CR-111, 121, 122, 125, 129)
	for i := range document.AdditionalDocumentReference {
		reference := &document.AdditionalDocumentReference[i]
		reference.UUID = ""
		if reference.Attachment != nil && reference.Attachment.ExternalReference != nil {
			reference.Attachment.ExternalReference = &ubl.ExternalReference{URI: reference.Attachment.ExternalReference.URI}
		}
	}

	return nil
}

// exportBindingDetails keeps delivery, charges and items to their EN 16931 business terms
func exportBindingDetails(_ *ubl.Format, _ *gobl.Envelope, doc ubl.Document) error {
	document, ok := doc.(*ubl.Invoice)
	if !ok {
		return nil
	}

	// BT-72 is the only delivery date, and the deliver-to address has no plot or coordinates (UBL-CR-347, 372, 379)
	for _, delivery := range document.Delivery {
		if delivery == nil {
			continue
		}
		delivery.LatestDeliveryDate = nil
		if delivery.DeliveryLocation != nil {
			trimAddress(delivery.DeliveryLocation.Address)
		}
	}

	// A document allowance or charge states no exemption reason (UBL-CR-480, 481)
	for _, charge := range document.AllowanceCharge {
		for _, category := range charge.TaxCategory {
			if category != nil {
				category.TaxExemptionReasonCode = nil
				category.TaxExemptionReason = nil
			}
		}
	}

	// Items: no manufacturer, the line seller, and attributes as text, BT-161 already writing out the quantity (UBL-CR-611, 619)
	for _, lines := range [][]ubl.InvoiceLine{document.InvoiceLines, document.CreditNoteLines} {
		for _, line := range lines {
			if line.Item == nil {
				continue
			}
			line.Item.ManufacturerParty = nil
			if line.Item.AdditionalItemProperty != nil {
				for i := range *line.Item.AdditionalItemProperty {
					(*line.Item.AdditionalItemProperty)[i].ValueQuantity = nil
				}
			}
		}
	}

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

// trimAddress drops the plot and the coordinates, which have no BT (UBL-CR-160, 167, 223, 230, 317, 324)
func trimAddress(address *ubl.PostalAddress) {
	if address == nil {
		return
	}

	address.PlotIdentification = nil
	address.LocationCoordinate = nil
}

// trimReference keeps a document reference's identifier and issue date, the only parts with a BT
func trimReference(reference *ubl.Reference) {
	reference.UUID = ""
	reference.DocumentTypeCode = ""
	reference.DocumentType = ""
	reference.Attachment = nil
}
