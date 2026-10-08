package converter

import (
	"bytes"
	"encoding/xml"

	cii "github.com/invopop/gobl.cii"
	addontools "github.com/invopop/gobl.ro.anaf"
	addon "github.com/invopop/gobl.ro.anaf/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// stated holds what the document says and the parsers do not keep
type stated struct {
	typeCode     cbc.Code                     // BT-3
	paymentMeans cbc.Code                     // BT-81
	fileNames    map[string]string            // BT-125 file names by attachment URL
	vat          map[currency.Code]num.Amount // BT-110 and BT-111 by currency
}

// ReverseConvert reads a CIUS-RO document ANAF delivered, in UBL or CII, into a GOBL invoice.
func ReverseConvert(data []byte) (*bill.Invoice, error) {
	// Read the document with the library of its syntax
	invoice, stated, err := read(data)
	if err != nil {
		return nil, err
	}

	// Restore what CIUS-RO and the parsers lose
	delocalize(invoice, stated)

	// Run the addon over the invoice
	err = invoice.Calculate()
	if err != nil {
		return nil, ErrConversion.WithCause(err).WithMsg("gobl calculation failed")
	}

	return invoice, nil
}

// read picks the parser from the namespace of the root element
func read(data []byte) (*bill.Invoice, *stated, error) {
	namespace, err := rootNamespace(data)
	if err != nil {
		return nil, nil, ErrUnreadable.WithCause(err).WithMsg("document is not xml")
	}

	switch namespace {
	case ubl.NamespaceUBLInvoice, ubl.NamespaceUBLCreditNote:
		return readUBL(data)
	case cii.NamespaceRSM:
		return readCII(data)
	}

	return nil, nil, ErrUnreadable.WithMsg("document in namespace %q is neither UBL nor CII", namespace)
}

// readUBL parses a UBL invoice or credit note
func readUBL(data []byte) (*bill.Invoice, *stated, error) {
	// Parse the XML
	parsed, err := ubl.Parse(data)
	if err != nil {
		return nil, nil, ErrUnreadable.WithCause(err).WithMsg("ubl document cannot be parsed")
	}

	document, ok := parsed.(*ubl.Invoice)
	if !ok {
		return nil, nil, ErrUnreadable.WithMsg("ubl document is not an invoice")
	}

	// Convert to GOBL as Romanian, whatever CIUS-RO version the document names
	env, err := document.Convert(ubl.WithContext(addontools.ContextCIUSRO))
	if err != nil {
		return nil, nil, ErrConversion.WithCause(err).WithMsg("ro xml to gobl conversion failed")
	}

	invoice, err := extract(env)
	if err != nil {
		return nil, nil, err
	}

	// Keep what the parser loses
	stated := &stated{
		fileNames: make(map[string]string),
		vat:       make(map[currency.Code]num.Amount),
	}

	if document.InvoiceTypeCode != nil {
		stated.typeCode = cbc.Code(document.InvoiceTypeCode.Value)
	}
	if document.CreditNoteTypeCode != nil {
		stated.typeCode = cbc.Code(document.CreditNoteTypeCode.Value)
	}

	if len(document.PaymentMeans) > 0 {
		stated.paymentMeans = cbc.Code(document.PaymentMeans[0].PaymentMeansCode.Value)
	}

	for _, reference := range document.AdditionalDocumentReference {
		if reference.Attachment != nil && reference.Attachment.ExternalReference != nil {
			stated.fileNames[reference.Attachment.ExternalReference.URI] = reference.Attachment.ExternalReference.FileName
		}
	}

	for _, total := range document.TaxTotal {
		amount, err := num.AmountFromString(total.TaxAmount.Value)
		if err == nil && total.TaxAmount.CurrencyID != nil {
			stated.vat[currency.Code(*total.TaxAmount.CurrencyID)] = amount
		}
	}

	return invoice, stated, nil
}

// readCII parses a CII invoice
func readCII(data []byte) (*bill.Invoice, *stated, error) {
	// Parse the XML
	document, err := cii.UnmarshalInvoice(data)
	if err != nil {
		return nil, nil, ErrUnreadable.WithCause(err).WithMsg("cii document cannot be parsed")
	}

	// Convert to GOBL
	env, err := cii.Parse(data)
	if err != nil {
		return nil, nil, ErrConversion.WithCause(err).WithMsg("ro xml to gobl conversion failed")
	}

	invoice, err := extract(env)
	if err != nil {
		return nil, nil, err
	}

	// gobl.cii cannot be told the document is Romanian
	invoice.Addons = tax.WithAddons(addon.V1)

	// Keep what the parser loses
	stated := &stated{
		vat: make(map[currency.Code]num.Amount),
	}

	if document.ExchangedDocument != nil {
		stated.typeCode = cbc.Code(document.ExchangedDocument.TypeCode)
	}

	if document.Transaction != nil && document.Transaction.Settlement != nil && len(document.Transaction.Settlement.PaymentMeans) > 0 {
		stated.paymentMeans = cbc.Code(document.Transaction.Settlement.PaymentMeans[0].TypeCode)
	}

	if document.Transaction != nil && document.Transaction.Settlement != nil && document.Transaction.Settlement.Summary != nil {
		for _, total := range document.Transaction.Settlement.Summary.TaxTotalAmount {
			amount, err := num.AmountFromString(total.Amount)
			if err == nil && total.Currency != "" {
				stated.vat[currency.Code(total.Currency)] = amount
			}
		}
	}

	return invoice, stated, nil
}

// rootNamespace returns the namespace of the first element of the document
func rootNamespace(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}

		if element, ok := token.(xml.StartElement); ok {
			return element.Name.Space, nil
		}
	}
}
