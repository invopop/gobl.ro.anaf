package efactura

import (
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

// StandardRates are the rates VAT category S may carry: 21% and 11% since
// August 2025, and 19%, 9% and 5% for the supplies made before it.
var StandardRates = []num.Percentage{
	num.MakePercentage(21, 2),
	num.MakePercentage(11, 2),
	num.MakePercentage(19, 2),
	num.MakePercentage(9, 2),
	num.MakePercentage(5, 2),
}

// exemptionCodes are the VATEX codes the guide gives the categories that have a
// single legal ground, used when the document states no code of its own.
var exemptionCodes = map[cbc.Code]cbc.Code{
	en16931.TaxCategoryOutsideScope:  "VATEX-EU-O",
	en16931.TaxCategoryReverseCharge: "VATEX-EU-AE",
}

// normalizeTaxCombo fills BT-121 for a supply not subject to VAT and for a
// domestic reverse charge, as the guide recommends in sections 3.3 and 3.4.
func normalizeTaxCombo(combo *tax.Combo) {
	if combo == nil || combo.Category != tax.CategoryVAT || combo.Ext.Has(cef.ExtKeyVATEX) {
		return
	}

	code, ok := exemptionCodes[comboCategory(combo)]
	if !ok {
		return
	}

	combo.Ext = combo.Ext.Set(cef.ExtKeyVATEX, code)
}

func taxComboRules() *rules.Set {
	return rules.For(new(tax.Combo),
		rules.When(
			is.Func("combo is standard rated VAT", comboIsStandard),
			rules.Assert("01", "standard rated VAT (S) must carry a Romanian rate (BT-152): 21% or 11%, or 19%, 9% or 5% for supplies made before August 2025",
				is.Func("romanian standard rate", comboHasStandardRate),
			),
		),
	)
}

// comboCategory returns the UNTDID 5305 code of a VAT combo, reading it off the
// key when the EN 16931 normalizer has not set the extension yet.
func comboCategory(combo *tax.Combo) cbc.Code {
	if code := combo.Ext.Get(untdid.ExtKeyTaxCategory); code != cbc.CodeEmpty {
		return code
	}

	switch combo.Key {
	case tax.KeyStandard:
		return en16931.TaxCategoryStandard
	case tax.KeyOutsideScope:
		return en16931.TaxCategoryOutsideScope
	case tax.KeyReverseCharge:
		return en16931.TaxCategoryReverseCharge
	case tax.KeyIntraCommunity:
		return en16931.TaxCategoryIntraCommunity
	}

	return cbc.CodeEmpty
}

func comboIsStandard(value any) bool {
	combo, ok := value.(*tax.Combo)

	return ok && combo != nil && combo.Category == tax.CategoryVAT && comboCategory(combo) == en16931.TaxCategoryStandard
}

func comboHasStandardRate(value any) bool {
	combo, ok := value.(*tax.Combo)
	if !ok || combo == nil || combo.Percent == nil {
		return false
	}

	for _, rate := range StandardRates {
		if combo.Percent.Compare(rate) == 0 {
			return true
		}
	}

	return false
}

// invoiceTaxSets returns every tax set of the breakdown: the lines, and the
// document level allowances and charges.
func invoiceTaxSets(invoice *bill.Invoice) []tax.Set {
	if invoice == nil {
		return nil
	}

	sets := make([]tax.Set, 0, len(invoice.Lines)+len(invoice.Discounts)+len(invoice.Charges))
	for _, line := range invoice.Lines {
		if line != nil {
			sets = append(sets, line.Taxes)
		}
	}
	for _, discount := range invoice.Discounts {
		if discount != nil {
			sets = append(sets, discount.Taxes)
		}
	}
	for _, charge := range invoice.Charges {
		if charge != nil {
			sets = append(sets, charge.Taxes)
		}
	}

	return sets
}

// invoiceVATCategories returns the distinct VAT categories of the breakdown.
func invoiceVATCategories(invoice *bill.Invoice) []cbc.Code {
	var codes []cbc.Code
	for _, set := range invoiceTaxSets(invoice) {
		for _, combo := range set {
			if combo == nil || combo.Category != tax.CategoryVAT {
				continue
			}
			if code := comboCategory(combo); !code.In(codes...) {
				codes = append(codes, code)
			}
		}
	}

	return codes
}

// IsOutsideScope reports an invoice not subject to VAT, category O, which in
// Romania only a seller not registered for VAT issues. Such an invoice states
// no VAT identifier for either party (BR-O-02, BR-O-04).
func IsOutsideScope(invoice *bill.Invoice) bool {
	return en16931.TaxCategoryOutsideScope.In(invoiceVATCategories(invoice)...)
}

// outsideScopeNotMixed applies BR-O-11: category O is the only one on the invoice.
func outsideScopeNotMixed(value any) bool {
	invoice, ok := value.(*bill.Invoice)
	if !ok || invoice == nil {
		return true
	}

	codes := invoiceVATCategories(invoice)

	return !en16931.TaxCategoryOutsideScope.In(codes...) || len(codes) == 1
}
