package local

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	columnCityName = "\u4e2d\u6587\u540d"
	columnAdcode   = "adcode"
)

var citySuffixes = []string{
	"\u5e02",
	"\u533a",
	"\u53bf",
	"\u81ea\u6cbb\u53bf",
	"\u81ea\u6cbb\u5dde",
	"\u5730\u533a",
	"\u76df",
	"\u65b0\u533a",
}

type CityCodeResolver struct {
	codes map[string]string
}

func LoadCityCodeResolver(path string) (*CityCodeResolver, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open adcode workbook: %w", err)
	}
	defer file.Close()

	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("adcode workbook has no sheets")
	}

	rows, err := file.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read adcode rows: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("adcode workbook is empty")
	}

	header := rows[0]
	nameIdx := -1
	adcodeIdx := -1
	for i, cell := range header {
		switch strings.TrimSpace(cell) {
		case columnCityName:
			nameIdx = i
		case columnAdcode:
			adcodeIdx = i
		}
	}
	if nameIdx < 0 || adcodeIdx < 0 {
		return nil, fmt.Errorf("required columns %s/%s not found", columnCityName, columnAdcode)
	}

	codes := make(map[string]string)
	for _, row := range rows[1:] {
		if nameIdx >= len(row) || adcodeIdx >= len(row) {
			continue
		}

		name := strings.TrimSpace(row[nameIdx])
		adcode := strings.TrimSpace(row[adcodeIdx])
		if name == "" || adcode == "" {
			continue
		}

		for _, key := range cityNameVariants(name) {
			codes[key] = adcode
		}
	}

	return &CityCodeResolver{codes: codes}, nil
}

func (r *CityCodeResolver) Resolve(city string) (string, bool) {
	for _, key := range cityNameVariants(city) {
		if adcode, ok := r.codes[key]; ok {
			return adcode, true
		}
	}
	return "", false
}

func cityNameVariants(name string) []string {
	raw := normalizeCityName(name)
	if raw == "" {
		return nil
	}

	seen := map[string]struct{}{raw: {}}
	variants := []string{raw}
	for _, suffix := range citySuffixes {
		if strings.HasSuffix(raw, suffix) {
			trimmed := strings.TrimSuffix(raw, suffix)
			if trimmed != "" {
				if _, ok := seen[trimmed]; !ok {
					seen[trimmed] = struct{}{}
					variants = append(variants, trimmed)
				}
			}
		}
	}

	return variants
}

func normalizeCityName(name string) string {
	replacer := strings.NewReplacer(" ", "", "\u3000", "", "\t", "", "\n", "", "\r", "")
	return replacer.Replace(strings.TrimSpace(name))
}
