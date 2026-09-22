package efactura

import (
	"regexp"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

// IdentityTypeCNP is the Romanian personal numeric code, BT-47.
const IdentityTypeCNP cbc.Code = "CNP"

// AnonymousCNP is what ANAF accepts when a consumer gives no code, OUG 138/2024.
const AnonymousCNP cbc.Code = "0000000000000"

// cnpRegexp is the shape of a personal numeric code.
var cnpRegexp = regexp.MustCompile(`^\d{13}$`)

var identities = []*cbc.Definition{
	{
		Code: IdentityTypeCNP,
		Name: i18n.String{
			i18n.EN: "Personal Numeric Code",
			i18n.RO: "Cod Numeric Personal",
		},
		Desc: i18n.String{
			i18n.EN: "Thirteen digit code identifying a Romanian natural person. Use thirteen zeros when the buyer of a consumer invoice gives no code.",
			i18n.RO: "Cod de treisprezece cifre care identifica o persoana fizica. Se folosesc treisprezece zerouri cand cumparatorul persoana fizica nu il comunica.",
		},
	},
}

// FindCNP returns the party's personal number, normalizing the identity type.
func FindCNP(party *org.Party) *org.Identity {
	if party == nil {
		return nil
	}

	for _, identity := range party.Identities {
		if identity != nil && cbc.NormalizeUpperCode(identity.Type) == IdentityTypeCNP {
			return identity
		}
	}

	return nil
}

func orgIdentityRules() *rules.Set {
	return rules.For(new(org.Identity),
		rules.When(
			org.IdentitiesTypeIn(IdentityTypeCNP),
			rules.Field("code",
				rules.Assert("01", "identity code for type CNP must be thirteen digits",
					is.MatchesRegexp(cnpRegexp),
				),
			),
		),
	)
}
