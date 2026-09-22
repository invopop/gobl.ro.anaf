package efactura

import (
	"testing"

	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
)

func TestNormalizeAddress(t *testing.T) {
	tests := []struct {
		name         string
		address      *org.Address
		wantRegion   string
		wantLocality string
	}{
		{
			name: "county in the state field",
			address: &org.Address{
				State:    "CJ",
				Locality: "Cluj-Napoca",
				Country:  l10n.RO.ISO(),
			},
			wantRegion:   "RO-CJ",
			wantLocality: "Cluj-Napoca",
		},
		{
			name: "county spelled out in the region field",
			address: &org.Address{
				Region:   "ro-cj",
				Locality: "Cluj-Napoca",
				Country:  l10n.RO.ISO(),
			},
			wantRegion:   "RO-CJ",
			wantLocality: "Cluj-Napoca",
		},
		{
			name: "bucharest sector",
			address: &org.Address{
				State:    "B",
				Locality: "Sector 1",
				Country:  l10n.RO.ISO(),
			},
			wantRegion:   "RO-B",
			wantLocality: "SECTOR1",
		},
		{
			name: "county written by name",
			address: &org.Address{
				Region:   "Cluj",
				Locality: "Cluj-Napoca",
				Country:  l10n.RO.ISO(),
			},
			wantRegion:   "RO-CJ",
			wantLocality: "Cluj-Napoca",
		},
		{
			name: "county name with diacritics",
			address: &org.Address{
				Region:   "Timiș",
				Locality: "Timisoara",
				Country:  l10n.RO.ISO(),
			},
			wantRegion:   "RO-TM",
			wantLocality: "Timisoara",
		},
		{
			name: "county name carrying its prefix",
			address: &org.Address{
				Region:   "Județul Bistrița-Năsăud",
				Locality: "Bistrita",
				Country:  l10n.RO.ISO(),
			},
			wantRegion:   "RO-BN",
			wantLocality: "Bistrita",
		},
		{
			name: "bucharest written by name",
			address: &org.Address{
				Region:   "București",
				Locality: "Sectorul 3",
				Country:  l10n.RO.ISO(),
			},
			wantRegion:   "RO-B",
			wantLocality: "SECTOR3",
		},
		{
			name: "unknown county is left for the rules to report",
			address: &org.Address{
				Region:   "Transylvania",
				Locality: "Cluj-Napoca",
				Country:  l10n.RO.ISO(),
			},
			wantRegion:   "Transylvania",
			wantLocality: "Cluj-Napoca",
		},
		{
			name: "foreign address is left alone",
			address: &org.Address{
				Region:   "Bavaria",
				Locality: "Munich",
				Country:  l10n.DE.ISO(),
			},
			wantRegion:   "Bavaria",
			wantLocality: "Munich",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			normalizeAddress(test.address)

			if test.address.Region != test.wantRegion {
				t.Errorf("region: want %s, got %s", test.wantRegion, test.address.Region)
			}
			if test.address.Locality != test.wantLocality {
				t.Errorf("locality: want %s, got %s", test.wantLocality, test.address.Locality)
			}
		})
	}
}
