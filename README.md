# GOBL Romania

GOBL add-on and converter for Romanian e-invoicing via RO e-Factura, the national system operated by ANAF (Agenția Națională de Administrare Fiscală).

## How it works

The add-on `ro-efactura-v1` registers itself with GOBL when its package is imported, and builds on the European `eu-en16931-v2017` add-on, which it pulls in automatically. An invoice that declares it in `$addons` gets the Romanian treatment whenever GOBL runs over it: `Calculate` quietly fixes what can be fixed, and `Validate` refuses what ANAF would refuse, before any XML exists. The Invopop apps work with invoices that already have the add-on applied.

```go
import (
	"github.com/invopop/gobl"
	_ "github.com/invopop/gobl.ro.anaf/addon"
)
```

The `ro-efactura-v1` key is not yet in GOBL core's registry of approved external add-ons, so it is not offered as a `$addons` value in the JSON Schema. A document declaring it fails validation with `add-on must be registered` unless this package is imported.

## Structure

```
addon/                The GOBL add-on (ro-efactura-v1): rules, normalizers, tags, document types
examples/             One invoice per case, with its calculated envelope under out/
resources/
├── artifacts/        The add-on map and the converter map
└── examples/         ANAF's official UBL and CII sample invoices
test/                 Conversion corpus of every invoice shape Romania accepts
```

The module root holds the converter and the tooling built on the add-on: `ContextCIUSRO` (the UBL context ANAF checks), `KindOf` (what ANAF does with an invoice), `Issuer` and `CIF` (who an invoice is filed under). The routing that decides how ANAF receives an invoice lives in gov-ro.

## Rules

The add-on adds 64 rules on top of GOBL and EN 16931, all under the `GOBL-RO-EFACTURA-` code prefix. Each one cites the ANAF rule it implements, such as `BR-RO-010`.

| Kind | Rules | What they check |
| --- | --- | --- |
| Required | 16 | Fields ANAF needs, such as the seller's tax number, the buyer's identifier and the street and city of every party |
| Code | 9 | Romanian codes and formats: document types, county and sector codes, the CNP, the VAT rates |
| Logic | 9 | Romanian invoice cases: corrections, credit notes, accounting invoices, self-billing, enforcement, category O |
| Length | 26 | The BR-RO-L character limits on names, addresses, notes, items and payment details |
| Count | 4 | The BR-RO-A limits on notes, preceding invoices, attachments and item attributes |

On top of the rules come 6 normalizers, 2 tags of its own (`accounting` and `enforcement`) and 5 document types (380, 381, 384, 389, 751). Every rule, normalizer, tag and document type is listed in the [add-on map](./resources/artifacts/ro-efactura-addon-map.html).

## Conversion

| Function | What it does |
| --- | --- |
| `ForwardConvert` | Takes a GOBL envelope, runs the add-on over it and returns CIUS-RO UBL ready to upload |
| `ReverseConvert` | Takes the XML of an invoice ANAF delivered, UBL or CII, and returns the GOBL invoice with the add-on applied |

The two directions mirror each other, so an invoice sent and read back comes out the same. The round trip, and how each Romanian detail is written and read back, are in the [converter map](./resources/artifacts/ro-efactura-converter-map.html).

## Testing

`go test ./...` is the whole check: the add-on's rules and normalizers, the examples, the converter, and the 205-invoice corpus in [`test`](./test/README.md), which drives every invoice through the add-on, the converter and ANAF's published schematron. The corpus needs phorm, which it looks for on `localhost:9090` and fails loudly without. Routing is tested in gov-ro.

| Command | What it does |
| --- | --- |
| `go test ./...` | Runs everything |
| `go test ./... -count=1` | Runs everything, ignoring Go's test cache |
| `go test ./addon/...` | Runs only the add-on's unit tests |
| `go test . -run TestExamples` | Checks the examples against their calculated envelopes |
| `go test . -run TestExamples -update` | Rewrites the examples' calculated envelopes after an intended change |
| `go test .` | Runs only the root tests: the converter, including the round trip and ANAF's samples, and the examples |
| `go test ./test/...` | Runs only the conversion corpus |
| `PHORM_URL=http://phorm:9090 go test ./...` | Points the corpus at phorm on another host or port |
| `RO_SKIP_SCHEMATRON=1 go test ./...` | Runs offline, proving the add-on and the converter agree but no longer that ANAF would accept the result |

### Picking one case

Corpus subtests are named after the fixture file, without the extension. Anchor a single fixture with `$`, or `ro_389_selfBilled` also matches `ro_389_selfBilledAccounting` and the rest of its family.

| Command | What it does |
| --- | --- |
| `go test ./test/... -run 'TestPipeline/ro_389_selfBilled$' -v` | Runs one fixture |
| `go test ./test/... -run 'TestPipeline/ro_381' -v` | Runs every credit note |
| `go test ./test/... -run 'TestPipeline/county' -v` | Runs the sweep of all 42 counties |
| `go test . -run 'TestReverseConvertRoundTrip' -v` | Reads every accepted invoice back to GOBL and renders it again |

### Changing the corpus

`test/out/` is rewritten on every run, so a change to the rendered XML shows up as a diff there. A new case is an envelope in `test/data/` plus its entry in `test/manifest.json`: the suite fails on a fixture with no entry and on an entry with no fixture, so no case can be added without saying what it proves. The format is in the [corpus README](./test/README.md).

## Before pushing

Run these three from the repo root. `gofmt -l` prints nothing when everything is formatted.

| Command | What it does |
| --- | --- |
| `go test ./...` | Runs every test |
| `gofmt -l .` | Lists any file that is not formatted |
| `golangci-lint run ./...` | Runs the linters over the whole project |
