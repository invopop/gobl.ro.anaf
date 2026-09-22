// Package efactura implements the Romanian e-Factura rules as a GOBL addon,
// adding the RO_CIUS restrictions on top of the EU EN 16931 CIUS profile.
package efactura

import (
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/norm"
	"github.com/invopop/gobl/pkg/here"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

const (
	// Namespace is the rules namespace for the Romanian e-Factura addon.
	Namespace rules.Code = "RO-EFACTURA"

	// Key identifies the Romanian e-Factura addon family.
	Key cbc.Key = "ro-efactura"

	// V1 covers RO_CIUS 1.0.1 and its 1.0.9 schematron.
	V1 cbc.Key = Key + "-v1"
)

func init() {
	tax.RegisterAddonDef(newV1Addon())
	rules.RegisterWithGuard(
		Key.String(),
		rules.GOBL.Add(Namespace),
		is.InContext(tax.AddonIn(V1)),
		billDeliveryRules(),
		billInvoiceRules(),
		orgAttachmentRules(),
		orgAttributeRules(),
		orgIdentityRules(),
		orgItemRules(),
		orgNoteRules(),
		orgPartyRules(),
		payInstructionsRules(),
		payTermsRules(),
	)
	norm.RegisterWithGuard(
		is.InContext(tax.AddonIn(V1)),
		norm.For(normalizeInvoice),
		norm.For(normalizeAddress),
	)
}

func newV1Addon() *tax.AddonDef {
	return &tax.AddonDef{
		Key: V1,
		Requires: []cbc.Key{
			en16931.V2017,
		},
		Name: i18n.String{
			i18n.EN: "Romania e-Factura (RO_CIUS)",
			i18n.RO: "România e-Factura (RO_CIUS)",
		},
		Description: i18n.String{
			i18n.EN: here.Doc(`
				Support for the Romanian national e-invoicing system,
				RO e-Factura, operated by ANAF. The addon applies the
				RO_CIUS restrictions of the EN 16931 semantic model:
				the document types ANAF files, the ISO 3166-2:RO
				subdivisions and Bucharest sector codes addresses have
				to carry, the VAT accounting currency, and the buyer
				identification a consumer invoice needs.

				Built on the EU EN 16931 CIUS profile, declared as a
				dependency.
			`),
			i18n.RO: here.Doc(`
				Suport pentru sistemul national de facturare
				electronica RO e-Factura, administrat de ANAF.
				Addon-ul aplica restrictiile RO_CIUS asupra modelului
				semantic EN 16931: tipurile de documente acceptate de
				ANAF, subdiviziunile ISO 3166-2:RO si codurile de
				sector ale Bucurestiului, moneda de contabilizare a
				TVA si identificarea cumparatorului persoana fizica.

				Construit peste profilul CIUS EU EN 16931, declarat ca
				dependenta.
			`),
		},
		Sources: []*cbc.Source{
			{
				Title: i18n.String{
					i18n.EN: "RO_CIUS Technical and Usage Specifications",
					i18n.RO: "Specificatii tehnice si de utilizare RO_CIUS",
				},
				URL: "https://mfinante.gov.ro/web/efactura/informatii-tehnice",
			},
			{
				Title: i18n.String{
					i18n.EN: "ANAF RO e-Factura",
					i18n.RO: "ANAF RO e-Factura",
				},
				URL: "https://www.anaf.ro/anaf/internet/ANAF/despre_anaf/strategii_anaf/proiecte_digitalizare/e.factura",
			},
		},
		Identities: identities,
		Tags:       tags,
		Scenarios:  scenarios,
	}
}
