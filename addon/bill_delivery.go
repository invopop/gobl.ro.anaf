package efactura

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

func billDeliveryRules() *rules.Set {
	return rules.For(new(bill.DeliveryDetails),
		rules.When(
			is.Func("delivery has an address", deliveryHasAddress),
			rules.Assert("01", "invoice delivery address line 1 (BT-75) is required when a delivery address (BG-15) is provided (BR-RO-180)",
				is.Func("delivery address has a street", deliveryAddressHasStreet),
			),
			rules.Assert("02", "invoice delivery city (BT-77) is required when a delivery address (BG-15) is provided (BR-RO-200)",
				is.Func("delivery address has a city", deliveryAddressHasLocality),
			),
			rules.Assert("03", "invoice delivery country subdivision (BT-79) is required when a delivery address (BG-15) is provided, whatever the country (BR-RO-210)",
				is.Func("delivery address has a region", deliveryAddressHasRegion),
			),
		),
	)
}

// deliveryAddress returns the receiver's first address, which is what gobl.ubl
// maps into the delivery location.
func deliveryAddress(value any) *org.Address {
	delivery, ok := value.(*bill.DeliveryDetails)
	if !ok || delivery == nil {
		return nil
	}

	return exportedAddress(delivery.Receiver)
}

func deliveryHasAddress(value any) bool {
	return deliveryAddress(value) != nil
}

func deliveryAddressHasStreet(value any) bool {
	return addressHasStreet(deliveryAddress(value))
}

func deliveryAddressHasLocality(value any) bool {
	return addressHasLocality(deliveryAddress(value))
}

func deliveryAddressHasRegion(value any) bool {
	return addressHasRegion(deliveryAddress(value))
}
