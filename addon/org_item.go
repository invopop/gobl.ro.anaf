package efactura

import (
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

// BR-27 needs no rule here: GOBL folds a negative price into a negative
// quantity and rejects the rest with GOBL-ORG-ITEM-02.

// Lengths and counts the RO_CIUS puts on an item and its attributes.
const (
	maxItemName           = 100
	maxItemDescription    = 200
	maxItemAttributes     = 50
	maxAttributeLabel     = 50
	maxAttributeValue     = 100
	attributeUnitSeparate = " "
)

func orgItemRules() *rules.Set {
	return rules.For(new(org.Item),
		rules.Field("name",
			rules.Assert("01", "item name (BT-153) must be no more than 100 characters long (BR-RO-L100)",
				is.RuneLength(0, maxItemName),
			),
		),
		rules.Field("description",
			rules.Assert("02", "item description (BT-154) must be no more than 200 characters long (BR-RO-L200)",
				is.RuneLength(0, maxItemDescription),
			),
		),
		rules.Field("attributes",
			rules.Assert("03", "item attributes (BG-32) must contain no more than 50 entries (BR-RO-A050)",
				is.Length(0, maxItemAttributes),
			),
		),
	)
}

func orgAttributeRules() *rules.Set {
	return rules.For(new(org.Attribute),
		rules.Field("label",
			rules.Assert("01", "item attribute name (BT-160) must be no more than 50 characters long (BR-RO-L050)",
				is.RuneLength(0, maxAttributeLabel),
			),
		),
		rules.Assert("02", "item attribute value (BT-161) must be no more than 100 characters long (BR-RO-L100)",
			is.Func("attribute value within length", attributeValueWithinLength),
		),
	)
}

// attributeValue renders BT-161 the way gobl.ubl does, preferring the amount
// with its unit over the plain text.
func attributeValue(attribute *org.Attribute) string {
	if attribute == nil {
		return ""
	}

	if attribute.Amount != nil {
		value := attribute.Amount.String()
		if attribute.Unit != "" {
			value += attributeUnitSeparate + string(attribute.Unit)
		}
		return value
	}

	return attribute.Text
}

func attributeValueWithinLength(value any) bool {
	attribute, ok := value.(*org.Attribute)
	if !ok || attribute == nil {
		return true
	}

	return len([]rune(attributeValue(attribute))) <= maxAttributeValue
}
