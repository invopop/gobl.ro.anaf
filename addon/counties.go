package efactura

import (
	"slices"
	"strconv"
	"strings"

	"github.com/invopop/gobl/l10n"
)

// County is a Romanian subdivision in ISO 3166-2:RO form, without the prefix.
type County string

// Every Romanian county.
const (
	CountyAlba           County = "AB"
	CountyArad           County = "AR"
	CountyArges          County = "AG"
	CountyBacau          County = "BC"
	CountyBihor          County = "BH"
	CountyBistritaNasaud County = "BN"
	CountyBotosani       County = "BT"
	CountyBrasov         County = "BV"
	CountyBraila         County = "BR"
	CountyBucharest      County = "B"
	CountyBuzau          County = "BZ"
	CountyCarasSeverin   County = "CS"
	CountyCalarasi       County = "CL"
	CountyCluj           County = "CJ"
	CountyConstanta      County = "CT"
	CountyCovasna        County = "CV"
	CountyDambovita      County = "DB"
	CountyDolj           County = "DJ"
	CountyGalati         County = "GL"
	CountyGiurgiu        County = "GR"
	CountyGorj           County = "GJ"
	CountyHarghita       County = "HR"
	CountyHunedoara      County = "HD"
	CountyIalomita       County = "IL"
	CountyIasi           County = "IS"
	CountyIlfov          County = "IF"
	CountyMaramures      County = "MM"
	CountyMehedinti      County = "MH"
	CountyMures          County = "MS"
	CountyNeamt          County = "NT"
	CountyOlt            County = "OT"
	CountyPrahova        County = "PH"
	CountySatuMare       County = "SM"
	CountySalaj          County = "SJ"
	CountySibiu          County = "SB"
	CountySuceava        County = "SV"
	CountyTeleorman      County = "TR"
	CountyTimis          County = "TM"
	CountyTulcea         County = "TL"
	CountyVaslui         County = "VS"
	CountyValcea         County = "VL"
	CountyVrancea        County = "VN"
)

// Counties is the full list, in the order the schematron writes it.
var Counties = []County{
	CountyAlba, CountyArges, CountyArad, CountyBucharest, CountyBacau,
	CountyBihor, CountyBistritaNasaud, CountyBraila, CountyBotosani,
	CountyBrasov, CountyBuzau, CountyCluj, CountyCalarasi, CountyCarasSeverin,
	CountyConstanta, CountyCovasna, CountyDambovita, CountyDolj, CountyGorj,
	CountyGalati, CountyGiurgiu, CountyHunedoara, CountyHarghita, CountyIlfov,
	CountyIalomita, CountyIasi, CountyMehedinti, CountyMaramures, CountyMures,
	CountyNeamt, CountyOlt, CountyPrahova, CountySibiu, CountySalaj,
	CountySatuMare, CountySuceava, CountyTulcea, CountyTimis, CountyTeleorman,
	CountyValcea, CountyVrancea, CountyVaslui,
}

// countyNames maps the folded Romanian name of a county onto its code, so a
// party that wrote "Judetul Bistrita-Nasaud" still reaches BT-39.
var countyNames = map[string]County{
	"alba":           CountyAlba,
	"arad":           CountyArad,
	"arges":          CountyArges,
	"bacau":          CountyBacau,
	"bihor":          CountyBihor,
	"bistritanasaud": CountyBistritaNasaud,
	"botosani":       CountyBotosani,
	"brasov":         CountyBrasov,
	"braila":         CountyBraila,
	"bucuresti":      CountyBucharest,
	"buzau":          CountyBuzau,
	"carasseverin":   CountyCarasSeverin,
	"calarasi":       CountyCalarasi,
	"cluj":           CountyCluj,
	"constanta":      CountyConstanta,
	"covasna":        CountyCovasna,
	"dambovita":      CountyDambovita,
	"dolj":           CountyDolj,
	"galati":         CountyGalati,
	"giurgiu":        CountyGiurgiu,
	"gorj":           CountyGorj,
	"harghita":       CountyHarghita,
	"hunedoara":      CountyHunedoara,
	"ialomita":       CountyIalomita,
	"iasi":           CountyIasi,
	"ilfov":          CountyIlfov,
	"maramures":      CountyMaramures,
	"mehedinti":      CountyMehedinti,
	"mures":          CountyMures,
	"neamt":          CountyNeamt,
	"olt":            CountyOlt,
	"prahova":        CountyPrahova,
	"satumare":       CountySatuMare,
	"salaj":          CountySalaj,
	"sibiu":          CountySibiu,
	"suceava":        CountySuceava,
	"teleorman":      CountyTeleorman,
	"timis":          CountyTimis,
	"tulcea":         CountyTulcea,
	"vaslui":         CountyVaslui,
	"valcea":         CountyValcea,
	"vrancea":        CountyVrancea,
}

// diacritics folds the Romanian alphabet onto plain ASCII.
var diacritics = strings.NewReplacer(
	"ă", "a", "â", "a", "î", "i", "ș", "s", "ş", "s", "ț", "t", "ţ", "t",
	"Ă", "a", "Â", "a", "Î", "i", "Ș", "s", "Ş", "s", "Ț", "t", "Ţ", "t",
)

// namePrefixes are the words a party writes in front of a county name.
var namePrefixes = []string{"judetul", "judet", "jud.", "jud", "municipiul", "mun."}

// How the SECTOR-RO code list of BR-RO-100 names a Bucharest sector.
const (
	sectorPrefix = "SECTOR"
	sectors      = 6
)

// ParseCounty accepts a code such as "CJ" or "ro-cj", or the county's own name,
// taking the first candidate that names a county.
func ParseCounty(candidates ...string) (County, bool) {
	for _, candidate := range candidates {
		code := strings.ToUpper(strings.TrimSpace(candidate))
		code = strings.TrimPrefix(code, l10n.RO.String()+"-")

		if county := County(code); slices.Contains(Counties, county) {
			return county, true
		}

		if county, ok := countyNames[foldName(candidate)]; ok {
			return county, true
		}
	}

	return "", false
}

// foldName reduces a written county to the key countyNames is indexed by.
func foldName(name string) string {
	folded := diacritics.Replace(strings.ToLower(strings.TrimSpace(name)))

	for _, prefix := range namePrefixes {
		if trimmed, ok := strings.CutPrefix(folded, prefix); ok {
			folded = strings.TrimSpace(trimmed)
			break
		}
	}

	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		return -1
	}, folded)
}

// Subentity renders a county the way BT-39, BT-54, BT-68 and BT-79 want it.
func Subentity(county County) string {
	return l10n.RO.String() + "-" + string(county)
}

// IsSubentity reports a value already written as an ISO 3166-2:RO code (BR-RO-110).
func IsSubentity(value string) bool {
	county, ok := ParseCounty(value)
	return ok && Subentity(county) == value
}

// ParseSector reads a Bucharest sector out of a city value, accepting
// "Sector 3", "sectorul 5" and a bare "6".
func ParseSector(city string) (int, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(city))
	trimmed = strings.TrimPrefix(trimmed, "sectorul")
	trimmed = strings.TrimPrefix(trimmed, "sector")
	trimmed = strings.TrimSpace(trimmed)

	sector, err := strconv.Atoi(trimmed)
	if err != nil || sector < 1 || sector > sectors {
		return 0, false
	}

	return sector, true
}

// Sector renders a sector as the SECTOR-RO code BR-RO-100 wants in BT-37,
// BT-52, BT-66 and BT-77.
func Sector(sector int) string {
	return sectorPrefix + strconv.Itoa(sector)
}

// IsSector reports a city value already written as a SECTOR-RO code.
func IsSector(city string) bool {
	sector, ok := ParseSector(city)
	return ok && Sector(sector) == city
}
