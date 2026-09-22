package efactura

import (
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

// BR-RO-L140 (BT-83) needs no rule here: gobl.ubl takes it from a cbc.Code,
// which GOBL already caps at 128 characters.

// Lengths the RO_CIUS puts on the payment block. Terms notes go into BT-20,
// not BT-22, which is why they repeat the note limit.
const (
	maxTermsNotes     = 300
	maxPaymentDetail  = 100
	maxAccountName    = 200
	maxCardHolderName = 200
)

func payTermsRules() *rules.Set {
	return rules.For(new(pay.Terms),
		rules.Field("notes",
			rules.Assert("01", "payment terms (BT-20) must be no more than 300 characters long (BR-RO-L300)",
				is.RuneLength(0, maxTermsNotes),
			),
		),
	)
}

func payInstructionsRules() *rules.Set {
	return rules.For(new(pay.Instructions),
		rules.Field("detail",
			rules.Assert("01", "payment means text (BT-82) must be no more than 100 characters long (BR-RO-L100)",
				is.RuneLength(0, maxPaymentDetail),
			),
		),
		rules.Field("credit_transfer",
			rules.Assert("02", "payment account name (BT-85) must be no more than 200 characters long (BR-RO-L200)",
				is.Func("account name within length", accountNameWithinLength),
			),
		),
		rules.Field("card",
			rules.Field("holder",
				rules.Assert("03", "payment card holder name (BT-88) must be no more than 200 characters long (BR-RO-L200)",
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
