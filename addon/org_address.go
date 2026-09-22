package efactura

import (
	"strings"

	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
)

// Lengths the RO_CIUS puts on address fields, BR-RO-L020 to BR-RO-L150.
const (
	maxAddressLineOne  = 150
	maxAddressLineTwo  = 100
	maxAddressLocality = 50
	maxAddressPostCode = 20
)

// normalizeAddress rewrites a Romanian address into the ISO 3166-2:RO county
// and, in Bucharest, the sector code BR-RO-100 and BR-RO-110 ask for.
func normalizeAddress(address *org.Address) {
	if address.Country != l10n.RO.ISO() {
		return
	}

	county, ok := ParseCounty(address.Region, address.State.String())
	if !ok {
		return
	}

	address.Region = Subentity(county)

	if county != CountyBucharest {
		return
	}

	if sector, ok := ParseSector(address.Locality); ok {
		address.Locality = Sector(sector)
	}
}

// exportedAddress returns the party's first address, the one gobl.ubl exports.
func exportedAddress(party *org.Party) *org.Address {
	if party == nil || len(party.Addresses) == 0 {
		return nil
	}

	return party.Addresses[0]
}

// addressesExported returns the first address of a slice, the only one exported.
func addressesExported(value any) *org.Address {
	addresses, ok := value.([]*org.Address)
	if !ok || len(addresses) == 0 {
		return nil
	}

	return addresses[0]
}

// addressInRomania reports an address ANAF applies its subdivision rules to.
func addressInRomania(address *org.Address) bool {
	return address != nil && address.Country == l10n.RO.ISO()
}

// addressHasStreet reports an address that fills BT-35, BT-50, BT-64 or BT-75.
func addressHasStreet(address *org.Address) bool {
	return address != nil && strings.TrimSpace(address.LineOne()) != ""
}

// addressHasLocality reports an address that fills BT-37, BT-52, BT-66 or BT-77.
func addressHasLocality(address *org.Address) bool {
	return address != nil && strings.TrimSpace(address.Locality) != ""
}

// addressHasRegion reports an address that fills BT-39, BT-54, BT-68 or BT-79.
func addressHasRegion(address *org.Address) bool {
	return address != nil && strings.TrimSpace(address.Region) != ""
}

// partyAddressHasStreet applies BR-RO-081 and BR-RO-082. A missing party passes,
// leaving it to the rule that requires it.
func partyAddressHasStreet(value any) bool {
	party, ok := value.(*org.Party)
	if !ok || party == nil {
		return true
	}

	return addressHasStreet(exportedAddress(party))
}

// partyAddressHasLocality applies BR-RO-091 and BR-RO-092.
func partyAddressHasLocality(value any) bool {
	party, ok := value.(*org.Party)
	if !ok || party == nil {
		return true
	}

	return addressHasLocality(exportedAddress(party))
}
