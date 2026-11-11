package roefactura

import (
	"errors"

	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl.ro.anaf/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

// ForwardConvert renders the CIUS-RO document of a GOBL envelope. The envelope
// has to declare the Romanian addon and come calculated and validated with it,
// as a silo entry does, so it is converted as it is. Like any GOBL converter,
// rendering adjusts the envelope in place, such as rounding its amounts.
func ForwardConvert(env *gobl.Envelope) ([]byte, error) {
	// Extract the invoice
	invoice, err := extract(env)
	if err != nil {
		return nil, err
	}

	// Check Romania files a document of its kind
	err = filable(invoice)
	if err != nil {
		return nil, err
	}

	// Convert to CIUS-RO
	document, err := ubl.ExportInvoice(env, ubl.WithFormat(FormatCIUSRO))
	if err != nil {
		// Keep the errors the export funcs raise
		var exportErr *Error
		if errors.As(err, &exportErr) {
			return nil, exportErr
		}

		return nil, ErrConversion.WithCause(err).WithMsg("gobl to ro xml conversion failed")
	}

	// Serialize XML
	xml, err := ubl.Encode(document)
	if err != nil {
		return nil, ErrConversion.WithCause(err).WithMsg("ro xml serialization failed")
	}

	return xml, nil
}

// extract opens the envelope and returns the invoice it carries.
func extract(env *gobl.Envelope) (*bill.Invoice, error) {
	if env == nil {
		return nil, ErrInvalidDocument.WithMsg("envelope is empty")
	}

	invoice, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, ErrInvalidDocument.WithMsg("gobl invoice structure is invalid")
	}

	// Only an invoice the Romanian addon normalized and validated is converted
	if !addon.V1.In(invoice.GetAddons()...) {
		return nil, ErrInvalidDocument.WithMsg("gobl invoice does not declare the %s addon", addon.V1)
	}

	return invoice, nil
}

// filable reports whether ANAF receives a document of this kind at all.
func filable(invoice *bill.Invoice) error {
	kind := KindOf(invoice)

	switch kind.Status {
	case StatusSkipped:
		return ErrSkipped.WithMsg("a document of kind %s is not reported to ANAF", kind.Name)
	case StatusUnsupported:
		return ErrUnsupported.WithMsg("a document of kind %s cannot be filed in Romania", kind.Name)
	}

	return nil
}
