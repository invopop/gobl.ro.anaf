# GOBL ➡️ RO e-Factura

Romanian e-Factura addon and tooling for [GOBL](https://github.com/invopop/gobl).

Its own Go module, `github.com/invopop/gobl.ro.anaf`, depending on nothing but
GOBL itself.

## What it is

RO e-Factura is the national e-invoicing system operated by ANAF. Documents are
UBL 2.1 following RO_CIUS, a CIUS of EN 16931, validated against the
`RO16931-rules.sch` schematron before ANAF accepts them.

Unlike a format converter, the `addon` subpackage is a true GOBL **addon**: it
registers tags, identities, scenarios, normalizers and validation rules into
GOBL's global registry, so a document is corrected and rejected inside GOBL
rather than by the schematron.

## Layout

- `addon/` — the GOBL addon (`ro-efactura-v1`): what Romania adds on top of the
  EN 16931 CIUS profile, which it declares as a dependency. Kept dependency
  light so importing it never pulls in conversion tooling.
- the module root — the tooling built on the addon: the UBL context that
  carries BT-24 and the schematrons, the document kinds ANAF files, and the
  party helpers the ANAF API needs.
- `examples/` — one document per case, with its calculated envelope.

The converter that renders CIUS-RO, and the routing that decides how ANAF
receives a document, are not here yet. The root package is where they land.

## Usage

Add a blank import of the addon so it registers itself, then use GOBL as
normal:

```go
import (
	"github.com/invopop/gobl"
	_ "github.com/invopop/gobl.ro.anaf/addon"
)
```

Declare `ro-efactura-v1` on the document and `Calculate` + `Validate` run the
full Romanian normalization and rules. `Requires` pulls `eu-en16931-v2017` in
automatically.

> **Note**: the `ro-efactura-v1` key is not yet in GOBL core's approved
> external-addon registry, so it is not offered as a `$addons` value in the
> JSON Schema. The runtime check is what matters: a document declaring it fails
> validation with `add-on must be registered` unless this package is imported.

## What the addon covers

The addon is the whole local-compliance gate: a document it accepts is one
gobl.ubl can render and ANAF's schematron will take.

### Normalizers

| What | Why |
| --- | --- |
| Rounding set to `currency` | BR-RO-Z2, so no amount is stated on more than two decimals |
| County resolved to an ISO 3166-2:RO code | BR-RO-110 and mirrors. Accepts `CJ`, `ro-cj`, and the county's own name with or without diacritics, including a `Judetul` prefix |
| Bucharest locality resolved to a `SECTOR-RO` code | BR-RO-100 and mirrors. Accepts `Sector 3`, `SECTOR3`, `sectorul 5`, `3` |
| Placeholder CNP for a Romanian consumer | BR-RO-120, OUG 138/2024. Only for a buyer established in Romania, so a foreign party is never given a Romanian personal code |

### Rules

| Area | Rules |
| --- | --- |
| Document types | BR-RO-020: 380, 384, 389 and 751 on an invoice, 381 on a credit note |
| Invoice number | BR-RO-010, BR-RO-L200: BT-1 carries a digit and stays under 200 characters |
| Accounting currency | BR-RO-030: an invoice in another currency needs a rate to RON |
| Rounding | BR-RO-Z2: the document currency states its amounts on two decimals at most |
| Parties | BR-RO-065, BR-RO-120: the seller has a tax number, the buyer a VAT or legal identifier |
| Addresses | BR-RO-080, BR-RO-090: street and city are required on the seller, the buyer and the ordering seller |
| Counties | BR-RO-110, BR-RO-170, BR-RO-210: BT-39, BT-54, BT-68 and BT-79 use ISO 3166-2:RO codes |
| Bucharest | BR-RO-100, BR-RO-160, BR-RO-200: in RO-B the city is a `SECTOR-RO` code |
| Delivery | BR-RO-180, BR-RO-200, BR-RO-210: a delivery address states BT-75, BT-77 and BT-79, the last one whatever the country |
| Repetition | BR-RO-A020, BR-RO-A050, BR-RO-A500 |
| Lengths | The BR-RO-L series on parties, contacts, addresses, items, item attributes, notes, payment details, attachments, the accounting reference and the preceding invoice number |
| Exemption reason | BR-RO-L100: BT-120 stays under 100 characters, which is tighter than a note |
| Intra-community | BR-IC-11, BR-IC-12: a category K supply states a delivery date or period and a delivery country |

Tags GOBL does not define itself are declared by the addon: `accounting` for a
751 document and `enforcement` for a forced execution procedure. Self-billing
wins over both, so BT-3 and the `autofactura` upload flag always agree.

## What it does not cover

- The Romanian **tax regime** (country, currency, VAT rates, CIF validation).
  That belongs in GOBL core and is on its way there; until it lands, gobl.ubl
  cannot derive the RON tax currency, so BT-6 and BT-111 have to be written by
  the converter.
- Rules GOBL enforces on its own. BT-24 comes from the UBL context, BR-27 never
  fires because GOBL folds a negative price into a negative quantity and rejects
  what is left, and the BT-12 to BT-18 length caps are unreachable because
  `cbc.Code` stops at 128 characters.
- Rules that only exist once the document is UBL, which are left to the
  schematron.

## Examples

`examples/` holds one document per case, with the calculated and validated
envelope under `examples/out/`. Regenerate the golden output after an
intentional change with:

```sh
go test . -run TestExamples -update
```

## Tests

```sh
go test ./...
```

That is the addon's own unit tests and the examples above. The full pipeline
corpus stays in gov-ro, where the converter and the routing can be driven over
it as well.
