# GOBL ➡️ RO e-Factura

Romanian e-Factura addon and tooling for [GOBL](https://github.com/invopop/gobl).

Its own Go module, `github.com/invopop/gobl.ro.anaf`, depending on GOBL and its
UBL and CII converters.

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
  party helpers the ANAF API needs: the bare `CIF`, and the `Issuer` a document
  is filed under (the enforcement body, the buyer when self-billed, or the
  supplier).
- `converter/` — GOBL to CIUS-RO UBL (`ForwardConvert`), and a received UBL or
  CII document back to GOBL (`ReverseConvert`), with the corpus that drives both
  under `converter/test/`.
- `examples/` — one document per case, with its calculated envelope.

The routing that decides how ANAF receives a document stays in gov-ro.

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
| Placeholder CNP for a consumer | BR-RO-120, OUG 138/2024. A buyer in Romania, or of unknown country, with no tax number and no legal identifier gets thirteen zeros. A foreign buyer must state its own identifier |
| `Autofactură` note on a buyer-issued invoice | Guide section 2.4, art. 319(20)(k). Added to BT-22 when the buyer issues in the supplier's name, not when a company bills a supply to itself |
| VATEX code for categories O and AE | Guide sections 3.3 and 3.4: `VATEX-EU-O` and `VATEX-EU-AE` when the document states no code |
| Enforcement body identifier | Communiqué on the enforcement register: the payee's CIF repeated as an identity, so it reaches BT-60 |

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
| Corrections | Guide sections 2.1 to 2.3: a 381, a 384 and a storno 380 (negative total) reference the invoice they correct, and every reference carries its number (BT-25) and issue date (BT-26) |
| Credit note signs | Guide section 2.2: credit note lines are always positive, a credit note is corrected by a positive 380 |
| Accounting invoice | Guide section 2.5: a 751 names the documents it is based on, such as the fiscal receipt, in BT-22 |
| Enforcement | Communiqué on the enforcement register: the enforcement body is the payee (BG-10) with a name and a CIF, and enforcement never combines with self-billing |
| Not subject to VAT | Guide section 3.4, BR-O-11: category O, reserved to sellers not registered for VAT, is the only category on its invoice |
| Standard rate | Guide section 3.1: category S carries 21% or 11%, or 19%, 9% or 5% for supplies made before August 2025 |

Tags GOBL does not define itself are declared by the addon: `accounting` for a
751 document and `enforcement` for a forced execution procedure. Self-billing
wins over `accounting`, so BT-3 and the `autofactura` upload flag always agree,
and is rejected together with `enforcement`.

A self-billed invoice whose supplier and customer share a CIF is a supply the
company bills to itself (`IsSelfSupply`); every other one is issued by the buyer
in the supplier's name (`IsBuyerIssued`), which is what the mention and the
upload flag are for.

An invoice in category O (`IsOutsideScope`) comes from a seller not registered
for VAT, so the converter states no VAT identifier for either party: the seller's
bare CIF goes to BT-32 and the buyer's code to BT-47 (BR-O-02, BR-O-04).

## What it does not cover

- The Romanian **tax regime** (country, currency, VAT rates, CIF validation).
  That belongs in GOBL core and is on its way there; until it lands, gobl.ubl
  cannot derive the RON tax currency, so BT-6 and BT-111 have to be written by
  the converter.
- Rules GOBL enforces on its own. BT-24 comes from the UBL context, BR-27 never
  fires because GOBL folds a negative price into a negative quantity and rejects
  what is left, and the BT-12 to BT-18 length caps are unreachable because
  `cbc.Code` stops at 128 characters. GOBL also limits attachments to the
  MIME types ANAF accepts (PDF, PNG, JPEG, CSV, XLSX and ODS).
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

That is the addon's own unit tests, the examples above, the converter tests and
the conversion corpus, which needs phorm (see `converter/test/README.md`). The
routing checks stay in gov-ro.
