// Official sources:
//
//   - Order MF 1366/2021: RO_CIUS, the Romanian e-invoice specification, and its national rules
//     https://static.anaf.ro/static/10/Anaf/legislatie/OMF_1366_2021.pdf
//   - Order MF 4092/2022: RO_CIUS 1.0.1 (art. 2) and the BR-RO table every rule cites
//     (art. 3 alin. (2))
//     https://static.anaf.ro/static/10/Anaf/legislatie/OMF_4092_2022.pdf
//   - ANAF schematron ro16931-ubl-1.0.9: the checks ANAF runs on every upload. Watch the
//     technical information page for new versions
//     https://mfinante.gov.ro/ro/web/efactura/informatii-tehnice
//
// Official rules this addon does not check, and why:
//
//   - BR-RO-001, specification identifier: written by the converter.
//   - BR-RO-040, VAT point date code: gobl.ubl only writes the allowed codes 3, 35 and 432.
//   - BR-RO-L100, address line 3: gobl.ubl does not write it.
//   - BR-RO-L140, remittance information: comes from a GOBL code, at most 128 characters.
//   - BR-RO-L200, contract, order and other document references: come from GOBL codes, at
//     most 128 characters.
//   - BR-RO-L1000, seller additional legal information: gobl.ubl does not write it.
//   - BR-RO-230, size of the attached files: GOBL attachments are links, no file is embedded.
//   - BR-RO-050, 060, 070, 240, 250, 260, 270: recommendations ANAF does not validate.
//     BR-RO-060 is applied through the self-billing mention.

// Package efactura implements the Romanian e-Factura rules as a GOBL addon,
// adding the RO_CIUS national rules (BR-RO) on top of the EU EN 16931 addon.
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
		billInvoiceRules(),
		billDeliveryRules(),
		billDiscountRules(),
		billChargeRules(),
		billLineDiscountRules(),
		billLineChargeRules(),
		orgPartyRules(),
		orgItemRules(),
		orgAttributeRules(),
		orgNoteRules(),
		orgAttachmentRules(),
		payTermsRules(),
		payInstructionsRules(),
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
				national rules (BR-RO) RO_CIUS adds to the EN 16931
				semantic model: the document types ANAF files, the
				ISO 3166-2:RO subdivisions and Bucharest sector codes
				addresses have to carry, the VAT accounting currency,
				the identifiers of the seller and the buyer, and the
				length and count limits of every element.

				Built on the EU EN 16931 CIUS profile, declared as a
				dependency.
			`),
			i18n.RO: here.Doc(`
				Suport pentru sistemul national de facturare
				electronica RO e-Factura, administrat de ANAF.
				Addon-ul aplica regulile nationale (BR-RO) adaugate de
				RO_CIUS modelului semantic EN 16931: tipurile de
				documente acceptate de ANAF, subdiviziunile ISO
				3166-2:RO si codurile de sector ale Bucurestiului,
				moneda de contabilizare a TVA, identificatorii
				vanzatorului si cumparatorului, precum si limitele de
				lungime si de aparitii ale fiecarui element.

				Construit peste profilul CIUS EU EN 16931, declarat ca
				dependenta.
			`),
		},
		Sources: []*cbc.Source{
			{
				Title: i18n.String{
					i18n.EN: "Order MF 1366/2021 approving RO_CIUS and the national operational rules",
					i18n.RO: "Ordinul MF 1366/2021 privind RO_CIUS si regulile operationale nationale",
				},
				URL: "https://static.anaf.ro/static/10/Anaf/legislatie/OMF_1366_2021.pdf",
			},
			{
				Title: i18n.String{
					i18n.EN: "Order MF 4092/2022 amending RO_CIUS (version 1.0.1, BR-RO table)",
					i18n.RO: "Ordinul MF 4092/2022 de modificare a RO_CIUS (versiunea 1.0.1, tabelul BR-RO)",
				},
				URL: "https://static.anaf.ro/static/10/Anaf/legislatie/OMF_4092_2022.pdf",
			},
			{
				Title: i18n.String{
					i18n.EN: "RO e-Factura technical information and schematron",
					i18n.RO: "Informatii tehnice RO e-Factura si schematron",
				},
				URL: "https://mfinante.gov.ro/ro/web/efactura/informatii-tehnice",
			},
		},
		Identities: identities,
		Tags:       tags,
		Scenarios:  scenarios,
	}
}
