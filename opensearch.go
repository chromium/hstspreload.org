package main

import (
	"fmt"
	"net/http"
)

func searchXML(origin string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")

		fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/">
  <ShortName>HSTS Preload</ShortName>
  <Description>HSTS Preload List Status and Eligibility</Description>
  <Tags>HSTS, HTTPS, security</Tags>
  <Contact>hstspreload@chromium.org</Contact>
  <Url type="text/html" method="GET" template="%s/?domain={searchTerms}"/>
  <Url type="application/x-suggestions+json" method="GET" template="%s/autocomplete?domain={searchTerms}" />
</OpenSearchDescription>
`,
			origin,
			origin)
	}
}
