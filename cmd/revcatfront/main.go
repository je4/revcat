package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/je4/revcat/v2/tools/client"
	"net/http"
)

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := &http.Client{}
	c := client.NewClient(httpClient, "https://localhost:8443/graphql", nil)
	entries, err := c.MediathekEntries(context.Background(), []string{"zotero2-2486551.TJDM3289"})
	if err != nil {
		panic(err)
	}
	for _, entry := range entries.GetMediathekEntries() {
		fmt.Printf("%+v\n", entry)
	}
}
