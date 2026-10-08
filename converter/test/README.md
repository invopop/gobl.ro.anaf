# Conversion corpus

The end-to-end conversion suite for Romanian e-Factura. Every fixture is a GOBL
envelope the way a silo entry arrives, and every one is driven through:

1. the **addon** normalizes and validates it,
2. the **converter** renders CIUS-RO,
3. **phorm** judges the result against ANAF's published schematron.

How ANAF receives each document is routing, which is tested in gov-ro.

## Layout

- `data/` — one envelope per case.
- `manifest.json` — what the pipeline must make of each one.
- `out/` — the rendered XML, rewritten on every run so a change shows as a diff.
- `anaf/` — ANAF's official UBL and CII samples, read back by the reverse conversion tests.

## Running it

```sh
go test ./converter/test/...
```

Phorm has to be reachable, because it is what validates each rendered document
against ANAF's published schematron. The suite looks on `http://localhost:9090`
and fails loudly, rather than passing quietly, when it is not there.

| Command | What it does |
| --- | --- |
| `go test ./converter/test/...` | The whole corpus |
| `go test ./converter/test/... -count=1` | The whole corpus, ignoring Go's test cache |
| `go test ./converter/test/... -v` | The same, naming every fixture as it passes |
| `PHORM_URL=http://phorm:9090 go test ./converter/test/...` | Phorm on another host or port |
| `RO_SKIP_SCHEMATRON=1 go test ./converter/test/...` | Run offline, without the schematron layer |

Skipping the schematron still proves the addon and the converter agree, but it
stops proving ANAF would accept the result. It is a convenience for working
offline, not a substitute.

### Picking one case

Subtests are named after the fixture file, without the extension:

```sh
go test ./converter/test/... -run 'TestPipeline/ro_389_selfBilled$' -v   # one fixture
go test ./converter/test/... -run 'TestPipeline/ro_381' -v               # every credit note
go test ./converter/test/... -run 'TestPipeline/county' -v               # the county sweep
go test ./converter/test/... -run 'TestPipeline/ro_380_long' -v          # every length rule
```

The `$` matters on a single fixture: without it `ro_389_selfBilled` also matches
`ro_389_selfBilledAccounting` and the rest of the family.

### Reading a failure

A fixture that should convert but does not prints the rule that stopped it. One
that converts but the schematron refuses prints the BR-RO finding, with the rule
id, the message and the XPath it fired on.

`out/` is rewritten on every run, so a change to the rendered XML shows up there
as a diff.

## The manifest

A fixture either states the rule that has to reject it:

```json
{ "file": "ro_380_badNumber.json", "note": "BT-1 carries no digit", "fault": "BILL-INVOICE-01" }
```

or the document type code (BT-3) its rendered document carries:

```json
{ "file": "ro_389_selfBilled.json", "note": "self-billed invoice", "code": "389" }
```

`fault` is the rule code without its `GOBL-RO-EFACTURA-` prefix.

Adding a file to `data/` without an entry here fails the suite, and so does an
entry naming a file that is gone.

## What the corpus covers

Every use case the ANAF guide to invoice type codes (Ghid cod facturi v2.9)
describes, each in the shapes that actually occur.

| Guide | Cases |
| --- | --- |
| 2.1 Invoice (380) | B2B, B2C, B2G, foreign buyers, partial and full storno against the invoice, a storno corrected by a positive 380, positive and negative price differences, one 380 cancelling and restating values (scenario 6) |
| 2.2 Credit note (381) | returns, a later discount, several invoices credited at once, a credit note corrected by a positive 380, self-billed, self-supply, enforcement, a seller not registered for VAT |
| 2.3 Corrective (384) | positive corrected values, original values negative and correct ones positive (scenario 1), several invoices, self-billed, consumer |
| 2.4 Self-billing (389) | issued by the buyer with the `Autofactură` mention added or already given, with a delivery, winning over the accounting tag, and self-supply, which carries neither the mention nor the `autofactura` flag |
| 2.5 Accounting (751) | based on a fiscal receipt, already paid by it, for a consumer, with a delivery, in EUR |
| 3 VAT categories | S at 21% and 11% and at the 19%, 9% and 5% of earlier supplies, E with a VATEX code or a reason, penalties and SGR deposits as E, AE with and without a reason, O for a seller not registered for VAT to a business, a consumer and an EU business, the travel agent margin scheme, intra-community supplies |
| 4 Technical storno | BT-1 with the `S` suffix, BT-25 and BT-26 carrying the upload index and its date |
| 5 General rules | several mentions in one BT-22, supporting documents, ordering references, payment means, due dates, advances, tax points, foreign currencies |
| Enforcement | filed under the enforcement body named as payee, for an invoice and a credit note |
| Technical notes | CPV codes on B2G lines, combined nomenclature codes on high-risk products, new constructions, holiday vouchers by card and by ticket, simplified invoices to an unidentified consumer, a foreign consumer with a CNP, and a foreign buyer with no identifier, which is rejected |

It also sweeps all 42 ISO 3166-2:RO counties and all six Bucharest sectors,
each written the way a customer writes it rather than as a code, which proves
the normalizer and ANAF's own code list agree.

The rejections exercise the addon rule by rule: one fixture per rule wherever a
document can reach it, among them the reference every correction needs, the
sign of a credit note, the note a 751 needs, the enforcement body, category O
mixed with others and the Romanian standard rates. Four rules stay unexercised
on purpose: `BILL-INVOICE-18` and `-19`, which guard a tax block the normalizer
always creates, and `-21` and `-22`, which guard a document type code the
scenarios always set. They are there for the day one of those assumptions
stops holding.
