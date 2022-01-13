package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
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
					return
				}

				cert, err := getLeafCertificate(u.Host)
				if err != nil {
					panic(err)
				}

				if ok := crypto.CertHasSAN(cert); !ok {
					fmt.Println("Invalid certificate:", endpoint.Interface, catalogEntry.Name)
				}
			}(catalogEntry, endpoint)
		}
	}
	wg.Wait()
}

func getLeafCertificate(host string) (*x509.Certificate, error) {
	log.Print(host)
	conn, err := tls.Dial("tcp", host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return conn.ConnectionState().PeerCertificates[0], nil
}
