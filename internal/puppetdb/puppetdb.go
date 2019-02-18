package puppetdb

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// PuppetDB stores informations used to connect to a PuppetDB
type PuppetDB struct {
	options *Options
	client  *http.Client
}

// Options contains the options used to connect to a PuppetDB
type Options struct {
	URL        string
	CertPath   string
	CACertPath string
	KeyPath    string
	SSLVerify  bool
}

// Node is a structure returned by a PuppetDB
type Node struct {
	Certname           string `json:"certname"`
	LatestReportStatus string `json:"latest_report_status"`
}

// NewClient creates a new PuppetDB client
func NewClient(options *Options) (p *PuppetDB, err error) {
	var transport *http.Transport

	puppetdbURL, err := url.Parse(options.URL)
	if err != nil {
		err = fmt.Errorf("failed to parse PuppetDB URL: %v", err)
		return
	}

	if puppetdbURL.Scheme != "http" && puppetdbURL.Scheme != "https" {
		err = fmt.Errorf("%s is not a valid http scheme", puppetdbURL.Scheme)
		return
	}

	if puppetdbURL.Scheme == "https" {
		// Load client cert
		cert, err := tls.LoadX509KeyPair(options.CertPath, options.KeyPath)
		if err != nil {
			err = fmt.Errorf("failed to load keypair: %s", err)
			return nil, err
		}

		// Load CA cert
		caCert, err := ioutil.ReadFile(options.CACertPath)
		if err != nil {
			err = fmt.Errorf("failed to load ca certificate: %s", err)
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// Setup HTTPS client
		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: !options.SSLVerify,
		}
		tlsConfig.BuildNameToCertificate()
		transport = &http.Transport{TLSClientConfig: tlsConfig}
	} else {
		transport = &http.Transport{}
	}

	p = &PuppetDB{
		client:  &http.Client{Transport: transport},
		options: options,
	}
	return
}

// Nodes returns the list of nodes
func (p *PuppetDB) Nodes() (nodes []Node, err error) {
	err = p.get("nodes", &nodes)
	if err != nil {
		err = fmt.Errorf("failed to get nodes: %s", err)
		return
	}
	return
}

func (p *PuppetDB) get(endpoint string, object interface{}) (err error) {
	base := strings.TrimRight(p.options.URL, "/")
	url := fmt.Sprintf("%s/v4/%s", base, endpoint)
	req, err := http.NewRequest("GET", url, strings.NewReader(""))
	if err != nil {
		err = fmt.Errorf("failed to build request: %s", err)
		return
	}
	resp, err := p.client.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to call API: %s", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response: %s", err)
		return
	}
	err = json.Unmarshal(body, object)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal: %s", err)
		return
	}
	return
}