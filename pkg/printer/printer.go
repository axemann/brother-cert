package printer

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// printer is a struct to interact with a remote Brother printer
type printer struct {
	httpClient     *http.Client
	baseUrl        string
	detectedModel  string
}

// PrinterConfig contains the information necessary to create a printer
// type which interfaces with a remote Brother printer
type Config struct {
	Hostname      string
	Password      string
	UserAgent     string
	UseHttp       bool
	InsecureHTTPS bool
}

// custom transport to add User-Agent
type printerTransport struct {
	userAgent     string
	insecureHTTPS bool
}

func (trans *printerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// always set user-agent
	req.Header.Set("User-Agent", trans.userAgent)

	if trans.insecureHTTPS {
		t := http.DefaultTransport.(*http.Transport)
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return t.RoundTrip(req)
	}

	return http.DefaultTransport.RoundTrip(req)
}

// NewPrinter creates a new printer from a PrinterConfig
func NewPrinter(cfg Config) (*printer, error) {
	baseUrl := "https://" + cfg.Hostname
	// http instead?
	if cfg.UseHttp {
		baseUrl = "http://" + cfg.Hostname
	}

	// make cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	p := &printer{
		httpClient: &http.Client{
			// disable redirect (POSTs return 301 and if client follows it loses the post response)
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Jar: jar,

			// set client timeout
			Timeout: 30 * time.Second,
			Transport: &printerTransport{
				userAgent:     cfg.UserAgent,
				insecureHTTPS: cfg.InsecureHTTPS,
			},
		},
		baseUrl: baseUrl,
	}

	// login & get cookie
	err = p.login(cfg.Password)
	if err != nil {
		return nil, err
	}

	// detect model for model-specific field names
	model, err := p.detectModel()
	if err != nil {
		// non-fatal: fall back to default model fields
		model = ""
	}
	p.detectedModel = model

	return p, nil
}
