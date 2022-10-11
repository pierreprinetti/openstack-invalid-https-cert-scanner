package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/catalog"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/openshift/library-go/pkg/crypto"
)

func main() {
	opts := clientconfig.ClientOpts{Cloud: os.Getenv("OS_CLOUD")}

	client, err := clientconfig.NewServiceClient("identity", &opts)
	if err != nil {
		panic(err)
	}

	pages, err := catalog.List(client).AllPages()
	if err != nil {
		panic(err)
	}

	catalogEntries, err := catalog.ExtractServiceCatalog(pages)
	if err != nil {
		panic(err)
	}

	invalidCertificateDetected := false
	var wg sync.WaitGroup
	for _, catalogEntry := range catalogEntries {
		for _, endpoint := range catalogEntry.Endpoints {
			wg.Add(1)
			go func(catalogEntry tokens.CatalogEntry, endpoint tokens.Endpoint) {
				defer wg.Done()

				u, err := url.Parse(endpoint.URL)
				if err != nil {
					panic(err)
				}

				if u.Scheme != "https" {
					fmt.Printf("PASS (not https): %s %s (%s)\n", endpoint.Interface, catalogEntry.Name, u.Host)
					return
				}

				if u.Port() == "" {
					u.Host = u.Host + ":443"
				}

				cert, err := getLeafCertificate(u.Host)
				if err != nil {
					panic(err)
				}

				if ok := crypto.CertHasSAN(cert); !ok {
					invalidCertificateDetected = true
					err := cert.VerifyHostname(u.Host)
					if isHostnameError := crypto.IsHostnameError(err); isHostnameError {
						fmt.Printf("INVALID: %s %s (%s) (isHostnameError)\n", endpoint.Interface, catalogEntry.Name, u.Host)
					} else {
						fmt.Printf("INVALID: %s %s (%s)\n", endpoint.Interface, catalogEntry.Name, u.Host)
					}
				} else {
					fmt.Printf("PASS: %s %s (%s)\n", endpoint.Interface, catalogEntry.Name, u.Host)
				}
			}(catalogEntry, endpoint)
		}
	}
	wg.Wait()
	if invalidCertificateDetected {
		fmt.Println("At least one invalid certificate was detected.")
		os.Exit(1)
	}
}

func getLeafCertificate(host string) (*x509.Certificate, error) {
	conn, err := tls.Dial("tcp", host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return conn.ConnectionState().PeerCertificates[0], nil
}
