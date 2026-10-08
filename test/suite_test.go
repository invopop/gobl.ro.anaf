// Package test drives the Romanian conversion over a corpus of invoices: the addon normalizes and validates, the converter renders CIUS-RO and the schematron judges the result
package test

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	addontools "github.com/invopop/gobl.ro.anaf"
	addon "github.com/invopop/gobl.ro.anaf/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/phorm"
)

const (
	dataDir      = "data"
	outDir       = "out"
	manifestFile = "manifest.json"
)

// faultPrefix is what the addon puts in front of every rule code it reports.
const faultPrefix = "GOBL-RO-EFACTURA-"

// fixture is one invoice and either the rule that has to reject it or the BT-3 its rendered document carries
type fixture struct {
	File  string `json:"file"`
	Note  string `json:"note"`
	Fault string `json:"fault,omitempty"`
	Code  string `json:"code,omitempty"`
}

// typeCode reads BT-3 back out of a rendered document, whichever element
// carries it.
type typeCode struct {
	InvoiceTypeCode    string `xml:"InvoiceTypeCode"`
	CreditNoteTypeCode string `xml:"CreditNoteTypeCode"`
}

// Value returns the code the document was filed under.
func (t typeCode) Value() string {
	if t.CreditNoteTypeCode != "" {
		return t.CreditNoteTypeCode
	}

	return t.InvoiceTypeCode
}

// phormURL is where the validation service lives. Override it when phorm is
// not on localhost: PHORM_URL=http://phorm:9090 go test ./...
func phormURL() string {
	if url := os.Getenv("PHORM_URL"); url != "" {
		return url
	}

	return "http://localhost:9090"
}

// skipSchematron lets the suite run without phorm, at the cost of proving only
// that the addon and the converter agree with each other.
func skipSchematron() bool {
	return os.Getenv("RO_SKIP_SCHEMATRON") != ""
}

func TestPipeline(t *testing.T) {
	fixtures := loadManifest(t)
	requireManifestComplete(t, fixtures)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("cannot create %s: %v", outDir, err)
	}

	client := phorm.New(phormURL(), "")

	for _, want := range fixtures {
		t.Run(strings.TrimSuffix(want.File, ".json"), func(t *testing.T) {
			envelope := loadEnvelope(t, want.File)
			invoice := invoiceOf(t, envelope)

			document, err := addontools.ForwardConvert(envelope)

			if want.Fault != "" {
				requireRejected(t, err, want.Fault)
				return
			}

			if err != nil {
				t.Fatalf("%s: conversion failed: %v", want.Note, err)
			}

			requireEnvelopeUntouched(t, envelope, invoice)
			writeOutput(t, want.File, document)

			if got := documentTypeCode(t, document); got != want.Code {
				t.Errorf("document type code: want %s, got %s", want.Code, got)
			}

			requireFilable(t, client, envelope, document)
		})
	}
}

// requireFilable asks phorm whether ANAF's own schematron takes the document.
func requireFilable(t *testing.T, client *phorm.Client, envelope *gobl.Envelope, document []byte) {
	t.Helper()

	if skipSchematron() {
		return
	}

	vesID := addontools.ContextCIUSRO.GetVESID(invoiceOf(t, envelope))

	validation, err := client.ValidateXml(context.Background(), &phorm.ValidateXmlRequest{Vesid: vesID, XmlContent: document})
	if err != nil {
		t.Fatalf("phorm at %s could not validate the document, set RO_SKIP_SCHEMATRON=1 to run without it: %v", phormURL(), err)
	}

	if !validation.Success {
		t.Errorf("the schematron rejected a document the addon accepted:\n%s", validation.Report())
	}
}

// requireEnvelopeUntouched checks the converter left the caller's document
// alone, because the silo entry is read again by the send action.
func requireEnvelopeUntouched(t *testing.T, envelope *gobl.Envelope, invoice *bill.Invoice) {
	t.Helper()

	if invoice.Tax != nil && invoice.Tax.Rounding != "" {
		t.Error("conversion wrote the rounding rule back into the caller's invoice")
	}

	if invoiceOf(t, envelope) != invoice {
		t.Error("conversion replaced the document the caller handed over")
	}
}

// requireRejected checks the addon turned the defect into a fault of its own,
// rather than leaving it for ANAF to find.
func requireRejected(t *testing.T, err error, fault string) {
	t.Helper()

	if err == nil {
		t.Fatalf("conversion succeeded, want fault %s", fault)
	}

	if !errors.Is(err, addontools.ErrNotCompliant) {
		t.Fatalf("conversion failed with the wrong error: %v", err)
	}

	if !strings.Contains(err.Error(), faultPrefix+fault) {
		t.Errorf("fault: want %s, got %v", faultPrefix+fault, err)
	}
}

// documentTypeCode reads BT-3 out of a rendered document.
func documentTypeCode(t *testing.T, document []byte) string {
	t.Helper()

	code := new(typeCode)

	if err := xml.Unmarshal(document, code); err != nil {
		t.Fatalf("cannot read the document type code: %v", err)
	}

	return code.Value()
}

// invoiceOf returns the invoice an envelope carries.
func invoiceOf(t *testing.T, envelope *gobl.Envelope) *bill.Invoice {
	t.Helper()

	invoice, ok := envelope.Extract().(*bill.Invoice)
	if !ok {
		t.Fatal("envelope carries no invoice")
	}

	return invoice
}

// loadEnvelope reads a fixture the way the silo hands it over: a document the
// caller already declared the Romanian addon on.
func loadEnvelope(t *testing.T, name string) *gobl.Envelope {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dataDir, name))
	if err != nil {
		t.Fatalf("cannot read fixture: %v", err)
	}

	envelope := new(gobl.Envelope)

	if err := json.Unmarshal(data, envelope); err != nil {
		t.Fatalf("fixture is not a GOBL envelope: %v", err)
	}

	if !addon.V1.In(invoiceOf(t, envelope).GetAddons()...) {
		t.Fatalf("fixture does not declare the %s addon", addon.V1)
	}

	return envelope
}

// writeOutput keeps the rendered XML, so a change shows up as a diff.
func writeOutput(t *testing.T, name string, document []byte) {
	t.Helper()

	if len(document) == 0 {
		t.Fatal("conversion produced no xml")
	}

	target := filepath.Join(outDir, strings.TrimSuffix(name, ".json")+".xml")

	if err := os.WriteFile(target, document, 0o644); err != nil { //nolint:gosec
		t.Fatalf("cannot write %s: %v", target, err)
	}
}

func loadManifest(t *testing.T) []fixture {
	t.Helper()

	data, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatalf("cannot read %s: %v", manifestFile, err)
	}

	var fixtures []fixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("%s is not a fixture list: %v", manifestFile, err)
	}

	return fixtures
}

// requireManifestComplete fails on a fixture with no entry in the manifest, so
// adding one to data forces a decision about what it proves, and on an entry
// naming a fixture that is no longer there.
func requireManifestComplete(t *testing.T, fixtures []fixture) {
	t.Helper()

	declared := make(map[string]bool, len(fixtures))
	for _, want := range fixtures {
		if declared[want.File] {
			t.Errorf("fixture %s is declared twice", want.File)
		}
		declared[want.File] = true

		if want.Fault == "" && want.Code == "" {
			t.Errorf("fixture %s declares neither a fault nor a document type code", want.File)
		}
	}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("cannot read %s: %v", dataDir, err)
	}

	present := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		present[entry.Name()] = true

		if !declared[entry.Name()] {
			t.Errorf("fixture %s has no entry in %s", entry.Name(), manifestFile)
		}
	}

	for _, want := range fixtures {
		if !present[want.File] {
			t.Errorf("%s names %s, which is not in %s", manifestFile, want.File, dataDir)
		}
	}
}
