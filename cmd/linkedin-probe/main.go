// Temporary probe — delete after confirming the correct LinkedIn Voyager endpoint.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ainsleyclark/godaily/pkg/env"
)

func jsessionFromCookie(cookie string) string {
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "JSESSIONID=") {
			v := strings.TrimPrefix(part, "JSESSIONID=")
			return strings.Trim(v, `"`)
		}
	}
	return ""
}

// Confirmed working URL from browser DevTools (count bumped to 20 for production).
const linkedInGraphQLURL = "https://www.linkedin.com/voyager/api/graphql?includeWebMetadata=true&variables=(start:0,origin:OTHER,query:(keywords:%23golang,flagshipSearchIntent:SEARCH_SRP,queryParameters:List((key:resultType,value:List(CONTENT))),includeFiltersInResponse:false),count:20)&queryId=voyagerSearchDashClusters.843215f2a3455f1bed85762a45d71be8"

func main() {
	cfg, err := env.New(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "env: %v\n", err)
		os.Exit(1)
	}
	if cfg.LinkedInCookie == "" {
		fmt.Fprintln(os.Stderr, "LINKEDIN_COOKIE is not set in .env")
		os.Exit(1)
	}

	csrf := jsessionFromCookie(cfg.LinkedInCookie)
	fmt.Fprintf(os.Stdout, "csrf-token: %q\n", csrf)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, linkedInGraphQLURL, nil)
	req.Header.Set("Cookie", cfg.LinkedInCookie)
	req.Header.Set("csrf-token", csrf)
	req.Header.Set("Accept", "application/vnd.linkedin.normalized+json+2.1")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")
	req.Header.Set("X-Li-Lang", "en_US")
	req.Header.Set("X-Li-Page-Instance", "urn:li:page:d_flagship3_search_srp_content;00000000-0000-0000-0000-000000000001")
	req.Header.Set("X-Li-Pem-Metadata", "Voyager - Content SRP=search-results")
	req.Header.Set("x-li-track", `{"clientVersion":"1.13.44541","mpVersion":"1.13.44541","osName":"web","timezoneOffset":1,"timezone":"Europe/London","deviceFormFactor":"DESKTOP","mpName":"voyager-web","displayDensity":2,"displayWidth":1920,"displayHeight":1080}`)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.linkedin.com/search/results/content/?keywords=%23golang")

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Fprintf(os.Stdout, "status: %d\n", resp.StatusCode)
	fmt.Fprintf(os.Stdout, "content-type: %s\n\n", resp.Header.Get("Content-Type"))

	out := "/tmp/linkedin_response.json"
	if err = os.WriteFile(out, body, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "full response written to %s (%d bytes)\n", out, len(body))
}
