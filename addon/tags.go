package efactura

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/tax"
)

// Tags Romania needs and the EN 16931 base profile does not define.
const (
	// TagEnforcement marks an invoice issued in a forced execution procedure.
	TagEnforcement cbc.Key = "enforcement"

	// TagAccounting marks an accounting-only invoice, UNTDID 1001 code 751.
	TagAccounting cbc.Key = "accounting"
)

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
					i18n.EN: "Invoice issued by a judicial enforcement officer in the name of the debtor.",
					i18n.RO: "Factura emisa de executorul judecatoresc in numele debitorului.",
				},
			},
		},
	},
}

// IsSelfBilled reports an invoice the buyer issued in the supplier's name.
func IsSelfBilled(invoice *bill.Invoice) bool {
	return invoice != nil && invoice.HasTags(tax.TagSelfBilled)
}

// IsEnforcement reports an invoice issued by a judicial enforcement officer.
func IsEnforcement(invoice *bill.Invoice) bool {
	return invoice != nil && invoice.HasTags(TagEnforcement)
}

// IsAccounting reports an invoice sent to ANAF for accounting purposes only.
func IsAccounting(invoice *bill.Invoice) bool {
	return invoice != nil && invoice.HasTags(TagAccounting)
}
