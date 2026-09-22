package roefactura

import (
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/cbc"

	efactura "github.com/invopop/gobl.ro.anaf/addon"
)

// ContextCIUSRO is the UBL context of the Romanian CIUS: the specification
// identifier ANAF checks in BT-24 (BR-RO-001), the addon that has to be
// declared to produce it, and the schematrons published for it. It moves to
// gobl.ubl the day that module learns about Romania.
var ContextCIUSRO = ubl.Context{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:efactura.mfinante.ro:CIUS-RO:1.0.1",
	Addons:          []cbc.Key{efactura.V1},
	VESIDs: ubl.VESIDMapping{
		Invoice:    "ro.gov.mfinante.cius-ro:ubl-invoice:1.0.9",
		CreditNote: "ro.gov.mfinante.cius-ro:ubl-creditnote:1.0.9",
	},
}
