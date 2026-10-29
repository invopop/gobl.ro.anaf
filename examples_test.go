package roefactura_test

import (
	"flag"
	"testing"

	"github.com/invopop/gobl/pkg/examples"

	// Register the Romanian e-Factura addon so the example documents
	// declaring ro-efactura-v1 normalize and validate.
	_ "github.com/invopop/gobl.ro.anaf/addon"
)

var update = flag.Bool("update", false, "update the example golden files")

// TestExamples converts every document under examples/ to a calculated,
// validated JSON envelope and compares it against its golden output, using the
// shared GOBL example helpers. Run with -update to regenerate the goldens.
func TestExamples(t *testing.T) {
	examples.Run(t, "examples", *update)
}
