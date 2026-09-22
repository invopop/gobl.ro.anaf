package efactura

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

// Document types ANAF accepts, BR-RO-020. Everything else the EN 16931 base
// profile can emit, such as 383 or 261, is rejected.
const (
	DocumentTypeInvoice    cbc.Code = "380"
	DocumentTypeCreditNote cbc.Code = "381"
	DocumentTypeCorrective cbc.Code = "384"
	DocumentTypeSelfBilled cbc.Code = "389"
	DocumentTypeAccounting cbc.Code = "751"
)

// DocumentTypes is the BR-RO-020_1 whitelist for BT-3 on an Invoice.
var DocumentTypes = []cbc.Code{
	DocumentTypeInvoice,
	DocumentTypeCorrective,
	DocumentTypeSelfBilled,
	DocumentTypeAccounting,
}

// CreditNoteTypes is the BR-RO-020_2 whitelist; in CIUS-RO a credit note is its
// own document, not an invoice with negative totals.
var CreditNoteTypes = []cbc.Code{
	DocumentTypeCreditNote,
}

// scenarios maps GOBL invoice types onto the UNTDID 1001 codes Romania files
// them under. Self-billing comes last so it wins over an accounting tag.
var scenarios = []*tax.ScenarioSet{
	{
		Schema: bill.ShortSchemaInvoice,
		List: []*tax.Scenario{
			{
				Types: []cbc.Key{bill.InvoiceTypeStandard},
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyDocumentType: DocumentTypeInvoice,
				}),
			},
			{
				Types: []cbc.Key{bill.InvoiceTypeCreditNote},
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyDocumentType: DocumentTypeCreditNote,
				}),
			},
			{
				Types: []cbc.Key{bill.InvoiceTypeCorrective},
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyDocumentType: DocumentTypeCorrective,
				}),
			},
			{
				Types: []cbc.Key{bill.InvoiceTypeStandard},
				Tags:  []cbc.Key{TagAccounting},
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyDocumentType: DocumentTypeAccounting,
				}),
			},
			{
				Types: []cbc.Key{bill.InvoiceTypeStandard},
				Tags:  []cbc.Key{tax.TagSelfBilled},
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyDocumentType: DocumentTypeSelfBilled,
				}),
			},
		},
	},
}
