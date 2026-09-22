package efactura

import (
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

// Lengths the RO_CIUS puts on a supporting document, BG-24.
const (
	maxAttachmentDescription = 100
	maxAttachmentURL         = 200
	maxAttachmentName        = 200
)

func orgAttachmentRules() *rules.Set {
	return rules.For(new(org.Attachment),
		rules.Field("description",
			rules.Assert("01", "supporting document description (BT-123) must be no more than 100 characters long (BR-RO-L100)",
				is.RuneLength(0, maxAttachmentDescription),
			),
		),
		rules.Field("url",
			rules.Assert("02", "external document location (BT-124) must be no more than 200 characters long (BR-RO-L200)",
				is.RuneLength(0, maxAttachmentURL),
			),
		),
		rules.Field("name",
			rules.Assert("03", "attached document filename (BT-125-2) must be no more than 200 characters long (BR-RO-L200)",
				is.RuneLength(0, maxAttachmentName),
			),
		),
	)
}
