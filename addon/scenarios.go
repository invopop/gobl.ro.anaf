// Scenarios and tags: the Romanian document type code of each invoice (BR-RO-020), the
// "Autofactură" mention when the buyer issues the invoice (BR-RO-060), and the enforcement
// tag (BR-RO-130).
//
// Official sources:
//
//   - Order MF 4092/2022: the BR-RO table of national rules (art. 3 alin. (2))
//     https://static.anaf.ro/static/10/Anaf/legislatie/OMF_4092_2022.pdf
//   - Law 227/2015 (Fiscal Code) art. 319 alin. (20) lit. k): an invoice the buyer issues in the
//     name of the supplier carries the mention "autofactură"
//     https://static.anaf.ro/static/10/Anaf/legislatie/Cod_fiscal_norme_2023.htm

package efactura

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/tax"
)

// Document types ANAF accepts, BR-RO-020.
const (
	DocumentTypeInvoice    cbc.Code = "380"
	DocumentTypeCreditNote cbc.Code = "381"
	DocumentTypeCorrective cbc.Code = "384"
	DocumentTypeSelfBilled cbc.Code = "389"
	DocumentTypeAccounting cbc.Code = "751"
)

// DocumentTypes are the BR-RO-020 codes for BT-3 on an Invoice.
var DocumentTypes = []cbc.Code{
	DocumentTypeInvoice,
	DocumentTypeCorrective,
	DocumentTypeSelfBilled,
	DocumentTypeAccounting,
}

// CreditNoteTypes are the BR-RO-020 codes for BT-3 on a CreditNote.
var CreditNoteTypes = []cbc.Code{
	DocumentTypeCreditNote,
}

// Tags Romania needs and the EN 16931 addon does not define.
const (
	// TagEnforcement marks an invoice issued in an enforcement procedure, BR-RO-130.
	TagEnforcement cbc.Key = "enforcement"

	// TagAccounting marks an invoice issued for accounting purposes only, BR-RO-020 code 751.
	TagAccounting cbc.Key = "accounting"
)

// SelfBillingNote is the mention Fiscal Code art. 319 alin. (20) lit. k) asks of an invoice the buyer issues in the supplier's name.
const SelfBillingNote = "Autofactură"

var tags = []*tax.TagSet{
	{
		Schema: bill.ShortSchemaInvoice,
		List: []*cbc.Definition{
			{
				Key: TagAccounting,
				Name: i18n.String{
					i18n.EN: "Accounting",
					i18n.RO: "Informare contabila",
				},
				Desc: i18n.String{
					i18n.EN: "Invoice sent for accounting purposes only, issued under UNTDID 1001 code 751.",
					i18n.RO: "Factura transmisa doar in scop contabil, emisa cu codul UNTDID 1001 751.",
				},
			},
			{
				Key: TagEnforcement,
				Name: i18n.String{
					i18n.EN: "Enforcement",
					i18n.RO: "Executare silita",
				},
				Desc: i18n.String{
					i18n.EN: "Invoice issued in an enforcement procedure, naming the enforcement body as payee.",
					i18n.RO: "Factura emisa in procedura de executare silita, cu organul de executare ca beneficiar.",
				},
			},
		},
	},
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
			{
				Tags:   []cbc.Key{tax.TagSelfBilled},
				Filter: isBuyerIssued,
				Note: &tax.Note{
					Key:  tax.TagSelfBilled,
					Text: SelfBillingNote,
				},
			},
		},
	},
}

// isBuyerIssued is the scenario filter of the self-billing mention.
func isBuyerIssued(doc any) bool {
	invoice, ok := doc.(*bill.Invoice)

	return ok && IsBuyerIssued(invoice)
}

// IsSelfBilled reports an invoice the buyer issued in the supplier's name.
func IsSelfBilled(invoice *bill.Invoice) bool {
	return invoice != nil && invoice.HasTags(tax.TagSelfBilled)
}

// IsSelfSupply reports a self-billed invoice a company issues to itself, for a
// supply it is deemed to make to itself (Fiscal Code art. 319 alin. (8)).
func IsSelfSupply(invoice *bill.Invoice) bool {
	return IsSelfBilled(invoice) && SameTaxNumber(invoice.Supplier, invoice.Customer)
}

// IsBuyerIssued reports a self-billed invoice the buyer issues in the supplier's
// name, the case that needs the autofactura mention and upload flag.
func IsBuyerIssued(invoice *bill.Invoice) bool {
	return IsSelfBilled(invoice) && !IsSelfSupply(invoice)
}

// IsEnforcement reports an invoice issued in an enforcement procedure.
func IsEnforcement(invoice *bill.Invoice) bool {
	return invoice != nil && invoice.HasTags(TagEnforcement)
}

// IsAccounting reports an invoice sent to ANAF for accounting purposes only.
func IsAccounting(invoice *bill.Invoice) bool {
	return invoice != nil && invoice.HasTags(TagAccounting)
}
