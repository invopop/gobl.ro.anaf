package roefactura

import (
	"bytes"
	"encoding/xml"

	cii "github.com/invopop/gobl.cii"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

// ReverseConvert reads a CIUS-RO document ANAF delivered, in UBL or CII, into a GOBL invoice with the addon applied.
// The import funcs of FormatCIUSRO and FormatCIUSROCII restore what the generic readers cannot know about Romania.
func ReverseConvert(data []byte) (*bill.Invoice, error) {
	namespace, err := rootNamespace(data)
	if err != nil {
		return nil, ErrUnreadable.WithCause(err).WithMsg("document is not xml")
	}

	// Pick the reader from the namespace of the root element
	switch namespace {
	case ubl.NamespaceUBLInvoice, ubl.NamespaceUBLCreditNote:
		return readUBL(data)
	case cii.NamespaceRSM:
		return readCII(data)
	}

	return nil, ErrUnreadable.WithMsg("document in namespace %q is neither UBL nor CII", namespace)
}

// readUBL reads a UBL invoice or credit note as Romanian, whatever CIUS-RO version the document names
func readUBL(data []byte) (*bill.Invoice, error) {
	// Parse the XML
	parsed, err := ubl.Decode(data)
	if err != nil {
		return nil, ErrUnreadable.WithCause(err).WithMsg("ubl document cannot be parsed")
	}

	document, ok := parsed.(*ubl.Invoice)
	if !ok {
		return nil, ErrUnreadable.WithMsg("ubl document is not an invoice")
	}

	// Convert to GOBL, running the import funcs and calculating with the addon
	env, err := ubl.Import(document, ubl.WithFormat(FormatCIUSRO))
	if err != nil {
		return nil, ErrConversion.WithCause(err).WithMsg("ro xml to gobl conversion failed")
	}

	return extract(env)
}

// readCII reads a CII invoice as Romanian
func readCII(data []byte) (*bill.Invoice, error) {
	// Parse the XML
	parsed, err := cii.Decode(data)
	if err != nil {
		return nil, ErrUnreadable.WithCause(err).WithMsg("cii document cannot be parsed")
	}

	document, ok := parsed.(*cii.Invoice)
	if !ok {
		return nil, ErrUnreadable.WithMsg("cii document is not an invoice")
	}

	// Convert to GOBL, running the import funcs and calculating with the addon
	env, err := cii.Import(document, cii.WithFormat(FormatCIUSROCII))
	if err != nil {
		return nil, ErrConversion.WithCause(err).WithMsg("ro xml to gobl conversion failed")
	}

	return extract(env)
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
