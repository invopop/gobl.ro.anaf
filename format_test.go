package roefactura_test

import (
	"os"
	"path/filepath"
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/convert"

	roefactura "github.com/invopop/gobl.ro.anaf"
)

func TestFormatsAreDetectedOnImport(t *testing.T) {
	t.Run("UBL", func(t *testing.T) {
		document, err := os.ReadFile(filepath.Join("test", "out", "ro_380_advance.xml"))
		if err != nil {
			t.Fatal(err)
		}

		result, err := convert.Import(document)
		if err != nil {
			t.Fatalf("import failed: %v", err)
		}

		if got := result.Format.Key; got != roefactura.FormatCIUSRO.Key {
			t.Errorf("format: want %s, got %s", roefactura.FormatCIUSRO.Key, got)
		}
	})

	t.Run("CII", func(t *testing.T) {
		converted, err := cii.ExportInvoice(calculatedFixture(t, "ro_380_advance.json"), cii.WithFormat(roefactura.FormatCIUSROCII))
		if err != nil {
			t.Fatalf("cii conversion failed: %v", err)
		}

		document, err := cii.Encode(converted)
		if err != nil {
			t.Fatalf("cii serialization failed: %v", err)
		}

		result, err := convert.Import(document)
		if err != nil {
			t.Fatalf("import failed: %v", err)
		}

		if got := result.Format.Key; got != roefactura.FormatCIUSROCII.Key {
			t.Errorf("format: want %s, got %s", roefactura.FormatCIUSROCII.Key, got)
		}
	})
}
