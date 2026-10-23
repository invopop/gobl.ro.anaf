// Rules for the payment terms and payment means of the invoice.
//
// Official sources:
//
//   - Order MF 4092/2022: the BR-RO table of national rules (art. 3 alin. (2))
//     https://static.anaf.ro/static/10/Anaf/legislatie/OMF_4092_2022.pdf
//
// Each rule names the official rule it applies at the end of its message, such as
// (BR-RO-010). The addon README maps every BR-RO rule to the fault code GOBL reports.

package efactura

import (
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

// Limits BR-RO-L100, L200 and L300 put on the payment block.
const (
	maxTermsNotes     = 300
	maxPaymentDetail  = 100
	maxAccountName    = 200
	maxCardHolderName = 200
)

func payTermsRules() *rules.Set {
	return rules.For(new(pay.Terms),
		rules.Field("notes",
			rules.Assert("01", "payment terms must be no more than 300 characters long (BR-RO-L300)",
				is.RuneLength(0, maxTermsNotes),
			),
		),
	)
}

func payInstructionsRules() *rules.Set {
	return rules.For(new(pay.Instructions),
		rules.Field("detail",
			rules.Assert("01", "payment means text must be no more than 100 characters long (BR-RO-L100)",
				is.RuneLength(0, maxPaymentDetail),
			),
		),
		rules.Field("credit_transfer",
			rules.Assert("02", "payment account name must be no more than 200 characters long (BR-RO-L200)",
				is.Func("account name within length", accountNameWithinLength),
			),
		),
		rules.Field("card",
			rules.Field("holder",
				rules.Assert("03", "payment card holder name must be no more than 200 characters long (BR-RO-L200)",
					is.RuneLength(0, maxCardHolderName),
				),
			),
		),
	)
}

// accountNameWithinLength covers BT-85, taken from the first credit transfer.
func accountNameWithinLength(value any) bool {
	transfers, ok := value.([]*pay.CreditTransfer)
	if !ok || len(transfers) == 0 || transfers[0] == nil {
		return true
	}

	return len([]rune(transfers[0].Name)) <= maxAccountName
}
