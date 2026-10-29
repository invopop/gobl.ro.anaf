package roefactura

import (
	cii "github.com/invopop/gobl.cii"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"

	efactura "github.com/invopop/gobl.ro.anaf/addon"
)

// specificationCIUSRO is the specification identifier ANAF checks in BT-24 (BR-RO-001)
const specificationCIUSRO = "urn:cen.eu:en16931:2017#compliant#urn:efactura.mfinante.ro:CIUS-RO:1.0.1"

// FormatCIUSRO is the UBL format of the Romanian CIUS: the specification
// identifier, the addon that has to be declared to produce it, the
// schematrons published for it, and the export and import funcs, in export.go
// and import.go, that handle what gobl.ubl cannot decide on its own.
var FormatCIUSRO = ubl.Format{
	Key:             "ubl+cius-ro-v1",
	Name:            i18n.NewString("UBL CIUS-RO"),
	CustomizationID: specificationCIUSRO,
	Addons:          []cbc.Key{efactura.V1},
	VESIDs: ubl.VESIDMapping{
		Invoice:    "ro.gov.mfinante.cius-ro:ubl-invoice:1.0.9",
		CreditNote: "ro.gov.mfinante.cius-ro:ubl-creditnote:1.0.9",
	},
	ExportFuncs: []ubl.ExportFunc{
		exportSchemaLocation,
		exportOrderReference,
		exportOutsideScope,
		exportSelfBillingNote,
	},
	ImportFuncs: []ubl.ImportFunc{
		importDocumentType,
		importOutsideScope,
		importParties,
		importPaymentMeans,
		importAttachmentNames,
		importTaxCurrency,
		importSelfBillingNote,
	},
}

// FormatCIUSROCII is the CII format of the Romanian CIUS, so the invoices
// read from CII declare the Romanian addon and get the same import funcs as
// UBL.
var FormatCIUSROCII = cii.Format{
	Key:         "cii+cius-ro-v1",
	Name:        i18n.NewString("CII CIUS-RO"),
	GuidelineID: specificationCIUSRO,
	Version:     cii.VersionD16B,
	Addons:      []cbc.Key{efactura.V1},
	ImportFuncs: []cii.ImportFunc{
		importCIIDocumentType,
		importCIIOutsideScope,
		importCIIParties,
		importCIIPaymentMeans,
		importCIITaxCurrency,
		importCIISelfBillingNote,
	},
}
