package main

import (
	"crypto/tls"
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

	var wg sync.WaitGroup
	for _, catalogEntry := range catalogEntries {
		for _, endpoint := range catalogEntry.Endpoints {
			wg.Add(1)
			go func(catalogEntry tokens.CatalogEntry, endpoint tokens.Endpoint) {
				fmt.Println(endpointOK(endpoint.URL), catalogEntry.Name, endpoint.Interface)
				wg.Done()
			}(catalogEntry, endpoint)
		}
	}
	wg.Wait()
}

func endpointOK(fullURL string) bool {
	u, err := url.Parse(fullURL)
	if err != nil {
		panic(err)
	}

	if u.Scheme != "https" {
		return true
	}
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", u.Host, conf)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	for _, cert := range conn.ConnectionState().PeerCertificates {
		if !crypto.CertHasSAN(cert) {
			return false
		}
	}
	return true
}
