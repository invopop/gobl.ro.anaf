# GOBL Romania e-Factura addon

The `ro-efactura-v1` addon applies the Romanian national rules (BR-RO) of RO_CIUS 1.0.1 on top of the EU EN 16931 addon, which it requires. Every rule cites the BR-RO rule it implements, so a fault can be traced back to the official table.

```go
import _ "github.com/invopop/gobl.ro.anaf/addon"
```

## Official sources

| Source | What it gives | Link |
| --- | --- | --- |
| Order MF 1366/2021 | RO_CIUS and the national operational rules | [PDF](https://static.anaf.ro/static/10/Anaf/legislatie/OMF_1366_2021.pdf) |
| Order MF 4092/2022 | RO_CIUS 1.0.1 (art. 2) and the BR-RO table (art. 3 alin. (2)) every rule cites | [PDF](https://static.anaf.ro/static/10/Anaf/legislatie/OMF_4092_2022.pdf) |
| Schematron ro16931-ubl-1.0.9 | The BR-RO checks ANAF runs on every upload | [zip](https://mfinante.gov.ro/static/10/eFactura/ro16931-ubl-1.0.9.zip) |
| Fiscal Code (Law 227/2015) art. 319 alin. (20) lit. k) | The `Autofactură` mention of an invoice the buyer issues | [ANAF](https://static.anaf.ro/static/10/Anaf/legislatie/Cod_fiscal_norme_2023.htm) |
| OUG 120/2021 art. 10^1 alin. (3), as amended by OUG 138/2024 | Thirteen zeros for a natural person with no tax code | [PDF](https://static.anaf.ro/static/10/Anaf/legislatie/OUG_138_2024.pdf) |

To keep the addon current, watch the [RO e-Factura technical information page](https://mfinante.gov.ro/ro/web/efactura/informatii-tehnice) for a new schematron (`ro16931-ubl-1.0.x.zip` and its `Informatii_ver` changelog), and the Ministry of Finance orders for a new RO_CIUS version. Compare the new BR-RO table with the one below.

## BR-RO rules

Each row is a rule of the Order MF 4092/2022 table. Codes are written without the `GOBL-RO-EFACTURA-` prefix every fault carries.

| BR-RO | Element | Requirement | GOBL |
| --- | --- | --- | --- |
| BR-RO-001 | BT-24 | Specification identifier of CIUS-RO 1.0.1 | Converter |
| BR-RO-010 | BT-1 | At least one digit | BILL-INVOICE-01 |
| BR-RO-020 | BT-3 | 380, 384, 389 or 751 on an invoice, 381 on a credit note | Scenarios, BILL-INVOICE-21, 22 |
| BR-RO-030 | BT-6 | RON when the invoice currency is not RON | BILL-INVOICE-03 |
| BR-RO-040 | BT-8 | 3, 35 or 432 | gobl.ubl writes no other code |
| BR-RO-050 | BT-10 | Recommended for a public buyer (N/A) | Not checked |
| BR-RO-060 | BT-22 | Mentions Romanian law requires (N/A) | Self-billing scenario note |
| BR-RO-065 | BT-31, BT-32, BT-63 | Seller VAT or tax registration identifier, or tax representative VAT identifier | BILL-INVOICE-10 |
| BR-RO-070 | BT-33 | Seller additional legal information (N/A) | Not checked |
| BR-RO-080 | BT-35, BT-50 | Seller and buyer address line 1 | BILL-INVOICE-11, 14 |
| BR-RO-090 | BT-37, BT-52 | Seller and buyer city | BILL-INVOICE-12, 15 |
| BR-RO-100 | BT-37, BT-52 | Bucharest sector when the subdivision is RO-B | ORG-PARTY-11 |
| BR-RO-110 | BT-39, BT-54 | ISO 3166-2:RO subdivision in Romania | ORG-PARTY-10 |
| BR-RO-120 | BT-47, BT-48 | Buyer legal registration or VAT identifier | BILL-INVOICE-13 |
| BR-RO-130 | BT-59, BT-61 | Enforcement body named as payee, with its legal registration identifier (N/A) | BILL-INVOICE-35 |
| BR-RO-140 | BT-64 | Tax representative address line 1 | BILL-INVOICE-16 |
| BR-RO-150 | BT-66 | Tax representative city | BILL-INVOICE-17 |
| BR-RO-160 | BT-66 | Tax representative Bucharest sector | ORG-PARTY-11 |
| BR-RO-170 | BT-68 | Tax representative ISO 3166-2:RO subdivision | ORG-PARTY-10 |
| BR-RO-180 | BT-75 | Delivery address line 1 | BILL-DELIVERYDETAILS-01 |
| BR-RO-200 | BT-77 | Delivery city, a Bucharest sector in RO-B | BILL-DELIVERYDETAILS-02, ORG-PARTY-11 |
| BR-RO-210 | BT-79 | Delivery subdivision, ISO 3166-2:RO in Romania | BILL-DELIVERYDETAILS-03, ORG-PARTY-10 |
| BR-RO-230 | BT-125 | Attached files at most 3 MB | GOBL attachments are links, the converter embeds no file |
| BR-RO-240, 250 | BT-158 | CPV code for a public buyer (N/A) | Not checked |
| BR-RO-260 | BT-127 | Line mentions Romanian law requires (N/A) | Not checked |
| BR-RO-270 | BT-85 | Payment account name for a public buyer (N/A) | Not checked |
| BR-RO-A020 | BG-1 | At most 20 notes, the autofactura mention included | BILL-INVOICE-05 |
| BR-RO-A050 | BG-24, BG-32 | At most 50 supporting documents, 50 attributes per item | BILL-INVOICE-08, ORG-ITEM-03 |
| BR-RO-A500 | BG-3 | At most 500 preceding invoices | BILL-INVOICE-06 |
| BR-RO-L020 | BT-38, 53, 67, 78 | 20 characters | ORG-PARTY-09 |
| BR-RO-L050 | BT-37, 52, 66, 77 | 50 characters | ORG-PARTY-08 |
| BR-RO-L050 | BT-160 | 50 characters | ORG-ATTRIBUTE-01 |
| BR-RO-L100 | BT-19 | 100 characters | BILL-INVOICE-09 |
| BR-RO-L100 | BT-36, 51, 65, 76 | 100 characters | ORG-PARTY-07 |
| BR-RO-L100 | BT-162, 163, 164, 165 | 100 characters | gobl.ubl writes no address line 3 |
| BR-RO-L100 | BT-41, 56 | 100 characters | ORG-PARTY-03 |
| BR-RO-L100 | BT-42, 57 | 100 characters | ORG-PARTY-04 |
| BR-RO-L100 | BT-43, 58 | 100 characters | ORG-PARTY-05 |
| BR-RO-L100 | BT-82 | 100 characters | PAY-INSTRUCTIONS-01 |
| BR-RO-L100 | BT-97 | 100 characters | BILL-DISCOUNT-01 |
| BR-RO-L100 | BT-104 | 100 characters | BILL-CHARGE-01 |
| BR-RO-L100 | BT-120 | 100 characters | BILL-INVOICE-20 |
| BR-RO-L100 | BT-123 | 100 characters | ORG-ATTACHMENT-01 |
| BR-RO-L100 | BT-133 | 100 characters | ORG-NOTE-02 |
| BR-RO-L100 | BT-139 | 100 characters | BILL-LINEDISCOUNT-01 |
| BR-RO-L100 | BT-144 | 100 characters | BILL-LINECHARGE-01 |
| BR-RO-L100 | BT-153 | 100 characters | ORG-ITEM-01 |
| BR-RO-L100 | BT-161 | 100 characters | ORG-ATTRIBUTE-02 |
| BR-RO-L140 | BT-83 | 140 characters | A GOBL code, at most 128 characters |
| BR-RO-L150 | BT-35, 50, 64, 75 | 150 characters | ORG-PARTY-06 |
| BR-RO-L200 | BT-1 | 200 characters | BILL-INVOICE-02 |
| BR-RO-L200 | BT-25 | 200 characters | BILL-INVOICE-07 |
| BR-RO-L200 | BT-27, 44, 59, 62, 70 | 200 characters | ORG-PARTY-01 |
| BR-RO-L200 | BT-28, 45 | 200 characters | ORG-PARTY-02 |
| BR-RO-L200 | BT-85 | 200 characters | PAY-INSTRUCTIONS-02 |
| BR-RO-L200 | BT-88 | 200 characters | PAY-INSTRUCTIONS-03 |
| BR-RO-L200 | BT-124, BT-125 | 200 characters | ORG-ATTACHMENT-02, 03 |
| BR-RO-L200 | BT-154 | 200 characters | ORG-ITEM-02 |
| BR-RO-L200 | BT-12 to BT-18, BT-122 | 200 characters | GOBL codes, at most 128 characters |
| BR-RO-L300 | BT-20 | 300 characters | PAY-TERMS-01 |
| BR-RO-L300 | BT-22, BT-127 | 300 characters | ORG-NOTE-01 |
| BR-RO-L1000 | BT-33 | 1000 characters | gobl.ubl writes no seller additional legal information |
| BR-RO-Z2 | BT-92 to BT-117, BT-131 to BT-142 | Two decimals | BILL-INVOICE-04 and the rounding normalizer |

N/A is the severity the table gives a rule ANAF does not validate.

## Normalizers

| Normalizer | What it does | Source |
| --- | --- | --- |
| Rounding | States every amount on the currency's two decimals | BR-RO-Z2 |
| Customer | Moves a consumer's CNP into BT-47, or gives a buyer in Romania with no identifier the thirteen zeros | BR-RO-120, OUG 120/2021 art. 10^1 alin. (3) |
| Address | Writes the county as its ISO 3166-2:RO code and a Bucharest locality as its sector | BR-RO-100, BR-RO-110 |

## Scenarios and tags

| Invoice | Tags | BT-3 | Note |
| --- | --- | --- | --- |
| `standard` | | 380 | |
| `standard` | `accounting` | 751 | |
| `standard` | `self-billed` | 389 | `Autofactură` when the buyer issues it |
| `credit-note` | any | 381 | `Autofactură` when the buyer issues it |
| `corrective` | any | 384 | `Autofactură` when the buyer issues it |

The scenarios come after the EN 16931 ones, so they override the codes BR-RO-020 does not accept, such as 261, 326, 386 or 393. The `enforcement` tag marks an invoice issued in an enforcement procedure, which names the enforcement body as payee (BR-RO-130).

## What the addon leaves to others

- **VAT rates and the CIF check** belong to the Romanian tax regime in GOBL core.
- **VAT categories and VATEX codes** belong to the EN 16931 addon. An exempt line, category O included, states its VATEX code or an exemption note itself.
- **EN 16931 rules**, such as BR-O-11 or BR-IC-11, belong to the EN 16931 addon. Until it covers them, ANAF's schematron reports them.
