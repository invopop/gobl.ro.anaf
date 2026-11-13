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

func TestValidPersonalNumber(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"1900101123457", true},  // CNP born 1 January 1990
		{"7900101123458", true},  // resident foreigner
		{"5040229123456", true},  // born 29 February 2004, a leap year
		{"9000000000015", true},  // NIF
		{"0000000000000", true},  // consumer with no number
		{"1900101123456", false}, // wrong check digit
		{"9000000000016", false}, // NIF with a wrong check digit
		{"1901301123455", false}, // no thirteenth month
		{"1900229123453", false}, // 29 February 1990
		{"0900101123455", false}, // no sex digit
		{"190010112345", false},  // twelve digits
		{"190010112345A", false}, // not a number
	}

	for _, test := range tests {
		if got := validPersonalNumber(test.code); got != test.want {
			t.Errorf("%s: want %t, got %t", test.code, test.want, got)
		}
	}
}

func TestValidCUI(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"12345674", true},
		{"18547290", true},
		{"12345675", false},    // wrong check digit
		{"012345674", false},   // leading zero
		{"12345678901", false}, // eleven digits
		{"J40/1/2015", false},  // trade register number
		{"", false},
	}

	for _, test := range tests {
		if got := validCUI(test.code); got != test.want {
			t.Errorf("%q: want %t, got %t", test.code, test.want, got)
		}
	}
}
