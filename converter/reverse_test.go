package converter_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"

	"github.com/invopop/gobl.ro.anaf/converter"
)

// lossy are the GOBL fields CIUS-RO has no place for, so a round trip cannot bring them back
var lossy = regexp.MustCompile(`^(` +
	`series|code|` + // BT-1 joins series and code
	`.*addresses\.\d+\.(street|num|state)|` + // BT-35 joins street and number, BT-39 keeps the county code only
	`preceding\.\d+\.(series|code|type|reason)|` + // BT-25 holds the joined number only
	`notes\.\d+\.key|` + // BT-22 has no key
	`.*people\.\d+\.name\.(given|surname)|` + // BT-41 is a single name
	`payment\.advances\.\d+\.(description|key|percent)|` + // BT-113 is a single amount
	`lines\.\d+\.item\.ext|` + // the unit code may or may not repeat the unit
	`lines\.\d+\.item\.attributes\.\d+\.(key|ext)|` + // BT-160 has no key
	`lines\.\d+\.item\.identities\.\d+\.type|` + // BT-157 has no type
	`delivery\.date|` + // gobl.ubl adds the period start as the delivery date
	`\$tags` + // enforcement and self-billed corrections live in the upload flags only
	`)$`)

// unrecoverable are the fixtures a round trip cannot rebuild at all
var unrecoverable = map[string]string{
	"ro_380_intraCommunity.json": "zero VAT leaves no exchange rate to recover",
}

func TestReverseConvertRoundTrip(t *testing.T) {
	for _, file := range acceptedFixtures(t) {
		t.Run(file, func(t *testing.T) {
			if reason, ok := unrecoverable[file]; ok {
				t.Skip(reason)
			}

			// GOBL to XML
			want := calculatedFixture(t, file)
			document, err := converter.ForwardConvert(want)
			if err != nil {
				t.Fatalf("forward conversion failed: %v", err)
			}

			// XML back to GOBL
			invoice, err := converter.ReverseConvert(document)
			if err != nil {
				t.Fatalf("reverse conversion failed: %v", err)
			}

			// The same GOBL, apart from what CIUS-RO cannot carry
			for _, diff := range diffJSON(t, want.Extract(), invoice) {
				t.Error(diff)
			}

			// The same XML again
			env, err := gobl.Envelop(invoice)
			if err != nil {
				t.Fatalf("invoice read back cannot be enveloped: %v", err)
			}

			again, err := converter.ForwardConvert(env)
			if err != nil {
				t.Fatalf("second forward conversion failed: %v", err)
			}

			if !bytes.Equal(document, again) {
				t.Error("the invoice read back renders a different document")
			}
		})
	}
}

func TestReverseConvertReadsTheANAFSamples(t *testing.T) {
	tests := []struct {
		file     string
		kind     cbc.Key
		currency currency.Code
		payable  string
		rate     string
	}{
		{file: "UBL/eInvoice_ex.xml", kind: bill.InvoiceTypeStandard, currency: currency.RON, payable: "41340576.71"},
		{file: "UBL/creditNote_ex.xml", kind: bill.InvoiceTypeCreditNote, currency: currency.SEK, payable: "500.00", rate: "2.4900"},
		{file: "CII/CII_example0.xml", kind: bill.InvoiceTypeStandard, currency: currency.DKK, payable: "4675.00"},
		{file: "CII/CII_example1.xml", kind: bill.InvoiceTypeStandard, currency: currency.EUR, payable: "250.33"},
		{file: "CII/CII_example2.xml", kind: bill.InvoiceTypeStandard, currency: currency.EUR, payable: "250.33"},
		{file: "CII/CII_example3.xml", kind: bill.InvoiceTypeStandard, currency: currency.DKK, payable: "1125.00"},
		{file: "CII/CII_example4.xml", kind: bill.InvoiceTypeStandard, currency: currency.DKK, payable: "4675.00"},
		{file: "CII/CII_example6.xml", kind: bill.InvoiceTypeStandard, currency: currency.DKK, payable: "4675.00"},
		{file: "CII/CII_example7.xml", kind: bill.InvoiceTypeStandard, currency: currency.SEK, payable: "3200.00"},
		{file: "CII/CII_example8.xml", kind: bill.InvoiceTypeStandard, currency: currency.SEK, payable: "3200.00"},
		{file: "CII/CII_example9.xml", kind: bill.InvoiceTypeStandard, currency: currency.EUR, payable: "177.87"},
		{file: "CII/CII_example10.xml", kind: bill.InvoiceTypeStandard, currency: currency.DKK, payable: "4675.00"},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("test", "anaf", test.file))
			if err != nil {
				t.Fatalf("cannot read sample: %v", err)
			}

			invoice, err := converter.ReverseConvert(data)
			if err != nil {
				t.Fatalf("reverse conversion failed: %v", err)
			}

			if invoice.Type != test.kind {
				t.Errorf("type: want %s, got %s", test.kind, invoice.Type)
			}

			if invoice.Currency != test.currency {
				t.Errorf("currency: want %s, got %s", test.currency, invoice.Currency)
			}

			// The payable amount stays what ANAF sealed
			if got := invoice.Totals.Payable.String(); got != test.payable {
				t.Errorf("payable: want %s, got %s", test.payable, got)
			}

			if test.rate != "" && (len(invoice.ExchangeRates) != 1 || invoice.ExchangeRates[0].Amount.String() != test.rate) {
				t.Errorf("exchange rate: want %s, got %v", test.rate, invoice.ExchangeRates)
			}
		})
	}
}

func TestReverseConvertKeepsATradeRegisterNumberOutsideTheVATScope(t *testing.T) {
	// A seller outside the VAT scope moves the buyer's CIF to BT-47, where other software may put a trade register number instead
	document, err := converter.ForwardConvert(calculatedFixture(t, "ro_380_notVatRegistered.json"))
	if err != nil {
		t.Fatalf("forward conversion failed: %v", err)
	}
	document = bytes.Replace(document, []byte("<cbc:CompanyID>87654329</cbc:CompanyID>"), []byte("<cbc:CompanyID>J40/123/2020</cbc:CompanyID>"), 1)

	invoice, err := converter.ReverseConvert(document)
	if err != nil {
		t.Fatalf("reverse conversion failed: %v", err)
	}

	if invoice.Customer.TaxID != nil {
		t.Errorf("customer tax id: want none, got %v", invoice.Customer.TaxID)
	}

	if len(invoice.Customer.Identities) != 1 || invoice.Customer.Identities[0].Code != "J40/123/2020" {
		t.Errorf("customer identities: want the trade register number, got %v", invoice.Customer.Identities)
	}
}

func TestReverseConvertDerivesTheExchangeRateFromCII(t *testing.T) {
	// A CII invoice in EUR stating its VAT in RON as well, BT-6 and BT-111
	converted, err := cii.ConvertInvoice(calculatedFixture(t, "ro_380_exchangeRate.json"))
	if err != nil {
		t.Fatalf("cii conversion failed: %v", err)
	}

	document, err := converted.Bytes()
	if err != nil {
		t.Fatalf("cii serialization failed: %v", err)
	}
	document = bytes.Replace(document, []byte("<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>"), []byte("<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode><ram:TaxCurrencyCode>RON</ram:TaxCurrencyCode>"), 1)
	document = bytes.Replace(document, []byte(`<ram:TaxTotalAmount currencyID="EUR">1050.00</ram:TaxTotalAmount>`), []byte(`<ram:TaxTotalAmount currencyID="EUR">1050.00</ram:TaxTotalAmount><ram:TaxTotalAmount currencyID="RON">5339.25</ram:TaxTotalAmount>`), 1)

	invoice, err := converter.ReverseConvert(document)
	if err != nil {
		t.Fatalf("reverse conversion failed: %v", err)
	}

	if len(invoice.ExchangeRates) != 1 || invoice.ExchangeRates[0].Amount.String() != "5.0850" {
		t.Errorf("exchange rate: want 5.0850, got %v", invoice.ExchangeRates)
	}

	if got := invoice.Totals.Tax.String(); got != "1050.00" {
		t.Errorf("vat: want 1050.00, got %s", got)
	}
}

func TestReverseConvertRejectsWhatIsNotAnInvoice(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "not xml", data: "not xml"},
		{name: "unknown namespace", data: `<Invoice xmlns="urn:example:invoice"/>`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := converter.ReverseConvert([]byte(test.data))

			if !errors.Is(err, converter.ErrUnreadable) {
				t.Errorf("error: want %v, got %v", converter.ErrUnreadable, err)
			}
		})
	}
}

// acceptedFixtures lists the corpus fixtures ANAF accepts
func acceptedFixtures(t *testing.T) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("test", "manifest.json"))
	if err != nil {
		t.Fatalf("cannot read manifest: %v", err)
	}

	var fixtures []struct {
		File  string `json:"file"`
		Fault string `json:"fault"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("manifest is not a fixture list: %v", err)
	}

	files := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		if fixture.Fault == "" {
			files = append(files, fixture.File)
		}
	}

	return files
}

// calculatedFixture reads a corpus fixture with the addon applied, which is what a round trip comes back to
func calculatedFixture(t *testing.T, file string) *gobl.Envelope {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("test", "data", file))
	if err != nil {
		t.Fatalf("cannot read fixture: %v", err)
	}

	env := new(gobl.Envelope)
	if err := json.Unmarshal(data, env); err != nil {
		t.Fatalf("fixture is not a GOBL envelope: %v", err)
	}

	if err := env.Calculate(); err != nil {
		t.Fatalf("fixture cannot be calculated: %v", err)
	}

	return env
}

// diffJSON lists the fields two documents disagree on, leaving out the lossy ones
func diffJSON(t *testing.T, want, got any) []string {
	t.Helper()

	var diffs []string
	compare("", toJSON(t, want), toJSON(t, got), &diffs)

	return diffs
}

// compare walks both documents, recording each field that differs
func compare(path string, want, got any, diffs *[]string) {
	if reflect.DeepEqual(want, got) || lossy.MatchString(path) {
		return
	}

	// Objects field by field
	wantObject, wantOK := want.(map[string]any)
	gotObject, gotOK := got.(map[string]any)
	if wantOK && gotOK {
		for key := range mergeKeys(wantObject, gotObject) {
			compare(join(path, key), wantObject[key], gotObject[key], diffs)
		}
		return
	}

	// Lists of the same length item by item
	wantList, wantOK := want.([]any)
	gotList, gotOK := got.([]any)
	if wantOK && gotOK && len(wantList) == len(gotList) {
		for i := range wantList {
			compare(join(path, fmt.Sprint(i)), wantList[i], gotList[i], diffs)
		}
		return
	}

	*diffs = append(*diffs, fmt.Sprintf("%s: want %v, got %v", path, want, got))
}

func toJSON(t *testing.T, value any) map[string]any {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("cannot marshal document: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("cannot unmarshal document: %v", err)
	}

	// Identifiers that are new on every read
	delete(out, "uuid")

	return out
}

func mergeKeys(left, right map[string]any) map[string]bool {
	keys := make(map[string]bool, len(left)+len(right))
	for key := range left {
		keys[key] = true
	}
	for key := range right {
		keys[key] = true
	}

	return keys
}

func join(path, key string) string {
	if path == "" {
		return key
	}

	return path + "." + key
}
