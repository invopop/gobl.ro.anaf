package efactura

import "testing"

func TestParseCounty(t *testing.T) {
	tests := []struct {
		name       string
		candidates []string
		want       County
		found      bool
	}{
		{name: "bare code", candidates: []string{"CJ"}, want: CountyCluj, found: true},
		{name: "lower case", candidates: []string{"cj"}, want: CountyCluj, found: true},
		{name: "prefixed", candidates: []string{"RO-CJ"}, want: CountyCluj, found: true},
		{name: "prefixed lower case", candidates: []string{" ro-cj "}, want: CountyCluj, found: true},
		{name: "bucharest", candidates: []string{"B"}, want: CountyBucharest, found: true},
		{name: "second candidate", candidates: []string{"Cluj County", "CJ"}, want: CountyCluj, found: true},
		{name: "written by name", candidates: []string{"Cluj"}, want: CountyCluj, found: true},
		{name: "name with diacritics", candidates: []string{"Timi\u0219"}, want: CountyTimis, found: true},
		{name: "name without diacritics", candidates: []string{"timis"}, want: CountyTimis, found: true},
		{name: "name with a county prefix", candidates: []string{"Jude\u021bul Cluj"}, want: CountyCluj, found: true},
		{name: "compound name", candidates: []string{"Bistri\u021ba-N\u0103s\u0103ud"}, want: CountyBistritaNasaud, found: true},
		{name: "two word name", candidates: []string{"Satu Mare"}, want: CountySatuMare, found: true},
		{name: "bucharest by name", candidates: []string{"Bucure\u0219ti"}, want: CountyBucharest, found: true},
		{name: "bucharest with a municipality prefix", candidates: []string{"Municipiul Bucuresti"}, want: CountyBucharest, found: true},
		{name: "unknown", candidates: []string{"Bavaria"}},
		{name: "empty", candidates: []string{""}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			county, found := ParseCounty(test.candidates...)

			if found != test.found {
				t.Fatalf("found: want %t, got %t", test.found, found)
			}
			if county != test.want {
				t.Errorf("county: want %s, got %s", test.want, county)
			}
		})
	}
}

func TestSubentity(t *testing.T) {
	if got := Subentity(CountyCluj); got != "RO-CJ" {
		t.Errorf("subentity: want RO-CJ, got %s", got)
	}

	if !IsSubentity("RO-B") {
		t.Error("RO-B must be an ISO 3166-2:RO subdivision")
	}

	if IsSubentity("cj") {
		t.Error("a bare lower case county is not an ISO 3166-2:RO subdivision")
	}
}

func TestParseSector(t *testing.T) {
	tests := []struct {
		city  string
		want  int
		found bool
	}{
		{city: "Sector 3", want: 3, found: true},
		{city: "SECTOR3", want: 3, found: true},
		{city: "sectorul 5", want: 5, found: true},
		{city: "6", want: 6, found: true},
		{city: "Sector 7"},
		{city: "Sector 0"},
		{city: "Bucuresti"},
	}

	for _, test := range tests {
		t.Run(test.city, func(t *testing.T) {
			sector, found := ParseSector(test.city)

			if found != test.found {
				t.Fatalf("found: want %t, got %t", test.found, found)
			}
			if sector != test.want {
				t.Errorf("sector: want %d, got %d", test.want, sector)
			}
		})
	}
}

func TestSector(t *testing.T) {
	if got := Sector(2); got != "SECTOR2" {
		t.Errorf("sector: want SECTOR2, got %s", got)
	}

	if !IsSector("SECTOR2") {
		t.Error("SECTOR2 must be a SECTOR-RO code")
	}

	if IsSector("Sector 2") {
		t.Error("an unnormalized sector is not a SECTOR-RO code")
	}
}
