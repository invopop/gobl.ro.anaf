// Package converter turns a GOBL envelope into the CIUS-RO XML document ANAF
// expects. It knows nothing about how that document is routed or delivered.
package converter

import (
	"encoding/json"

	"github.com/invopop/gobl"
	addontools "github.com/invopop/gobl.ro.anaf"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

// ForwardConvert renders the CIUS-RO document of a GOBL envelope. The envelope is
// expected to declare the Romanian addon already, so that its contents are
// the ones ANAF is filed with. The caller's envelope is never modified.
func ForwardConvert(env *gobl.Envelope) ([]byte, error) {

	//***** Delete once the addon is implemeted

	// Work on a copy, because preparing the document rewrites it
	working, err := clone(env)
	if err != nil {
		return nil, err
	}

	//***** Delete once the addon is implemeted

	// Extract the invoice and check Romania files a document of its kind
	invoice, err := extract(working)
	if err != nil {
		return nil, err
	}

	err = filable(invoice)
	if err != nil {
		return nil, err
	}

	//***** Delete once the addon is implemeted

	// Run the addon over the invoice
	err = prepare(working)
	if err != nil {
		return nil, err
	}

	// Read the normalized invoice back, so localize sees what was rendered
	invoice, err = extract(working)
	if err != nil {
		return nil, err
	}

	//***** Delete once the addon is implemeted

	// Convert to CIUS-RO
	document, err := ubl.ConvertInvoice(working, ubl.WithContext(addontools.ContextCIUSRO))
	if err != nil {
		return nil, ErrConversion.WithCause(err).WithMsg("gobl to ro xml conversion failed")
	}

	// Apply what gobl.ubl cannot decide on its own yet
	err = localize(document, invoice)
	if err != nil {
		return nil, err
	}

	// Serialize XML
	xml, err := ubl.Bytes(document)
	if err != nil {
		return nil, ErrConversion.WithCause(err).WithMsg("ro xml serialization failed")
	}

	return xml, nil
}

// clone copies an envelope, so preparing the document leaves the caller's own
// copy untouched.
func clone(env *gobl.Envelope) (*gobl.Envelope, error) {
	if env == nil {
		return nil, ErrInvalidDocument.WithMsg("envelope is empty")
	}

	data, err := json.Marshal(env)
	if err != nil {
		return nil, ErrInvalidDocument.WithCause(err).WithMsg("envelope cannot be copied")
	}

	copied := new(gobl.Envelope)

	err = json.Unmarshal(data, copied)
	if err != nil {
		return nil, ErrInvalidDocument.WithCause(err).WithMsg("envelope cannot be copied")
	}

	return copied, nil
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

	return invoice, nil
}

// filable reports whether ANAF receives a document of this kind at all.
func filable(invoice *bill.Invoice) error {
	kind := addontools.KindOf(invoice)

	switch kind.Status {
	case addontools.StatusSkipped:
		return ErrSkipped.WithMsg("a document of kind %s is not reported to ANAF", kind.Name)
	case addontools.StatusUnsupported:
		return ErrUnsupported.WithMsg("a document of kind %s cannot be filed in Romania", kind.Name)
	}

	return nil
}

// prepare runs the addon over the invoice. The silo entry will eventually
// reach us already normalized and validated, at which point prepare can be deleted.
func prepare(env *gobl.Envelope) error {
	err := env.Calculate()
	if err != nil {
		return ErrConversion.WithCause(err).WithMsg("gobl calculation failed")
	}

	err = env.Validate()
	if err != nil {
		return ErrNotCompliant.WithCause(err).WithMsg("gobl invoice does not satisfy the romanian rules")
	}

	return nil
}
