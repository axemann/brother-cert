package printer

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
)

// ModelFields holds the form field names used by a specific Brother printer
// model's certificate import page. These obfuscated names (Bxxxx) vary by
// model, so they must be discovered per model (e.g. via mitmproxy).
type ModelFields struct {
	PageID                       string
	CSRFToken                    string // field name; value parsed from page
	FieldA                       string // first empty hidden field (e.g. B18a8)
	FieldB                       string // second empty hidden field (e.g. B18b6)
	CertProcessControl           string // hidden_certificate_process_control
	FileField                    string // multipart file field (e.g. B178f)
	Password                     string // password input field (e.g. B1790)
	HiddenCertImportPassword     string // hidden_cert_import_password
}

// knownModels maps a detected model name (e.g. "MFC-L3780CDW") to its
// certificate import field names.
var knownModels = map[string]ModelFields{
	"MFC-L2710DW": {
		PageID:                   "390",
		FieldA:                   "B8ea",
		FieldB:                   "B8f8",
		CertProcessControl:       "hidden_certificate_process_control",
		FileField:                "B820",
		Password:                 "B821",
		HiddenCertImportPassword: "hidden_cert_import_password",
	},
	"MFC-L3780CDW": {
		PageID:                   "507",
		FieldA:                   "B18a8",
		FieldB:                   "B18b6",
		CertProcessControl:       "hidden_certificate_process_control",
		FileField:                "B178f",
		Password:                 "B1790",
		HiddenCertImportPassword: "hidden_cert_import_password",
	},
}

// defaultModelFields is used when the model is not in knownModels.
var defaultModelFields = knownModels["MFC-L2710DW"]

// detectModel fetches the printer's status page and parses the model name.
func (p *printer) detectModel() (string, error) {
	u, err := url.ParseRequestURI(p.baseUrl)
	if err != nil {
		return "", err
	}
	u.Path = "/home/status.html"

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// parse from title: <title>Brother MFC-L3780CDW series</title>
	re := regexp.MustCompile(`Brother\s+(MFC-[A-Za-z0-9]+)`)
	m := re.FindSubmatch(bodyBytes)
	if len(m) < 2 {
		return "", fmt.Errorf("printer: could not detect model from status page")
	}
	return string(m[1]), nil
}

// modelFields returns the field names for this printer's model.
func (p *printer) modelFields() ModelFields {
	if p.detectedModel == "" {
		return defaultModelFields
	}
	if mf, ok := knownModels[p.detectedModel]; ok {
		return mf
	}
	return defaultModelFields
}
