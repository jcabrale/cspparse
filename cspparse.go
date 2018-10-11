package main

/*
-====== CSP Parser ======-
by: Corben Leo (https://corben.io)
Contributor(s):
- Riley Johnson (https://therileyjohnson.com)

%% Description:
> Gets Content-Security-Policies for given URL / Domain.
> Output is in ReconJSON (https://github.com/ReconJSON/ReconJSON)
*/

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/imroc/req"
	"github.com/pkg/errors"
)

// create a struct (a collection of fields) to
// "unravel" Google's API JSON response into
type cspStatus struct {
	Status string `json:"status"`
	Csp    string `json:"csp"`
}

// define a map of lists (with type string)
// globally so any function can access it
var cspObject map[string]interface{}

func main() {
	// initialize the cspObject map
	cspObject = make(map[string]interface{})

	// adding objects to the map cspObject, this
	// specific data is to make the output valid ReconJSON
	cspObject["type"] = "ServiceDescriptor"
	cspObject["name"] = "httpCsp"
	cspObject["location"] = "Header"

	// if user passes a command line argument (the domain  / url to check)
	if len(os.Args) > 1 {

		// set the variable to the first argument passed
		domain := os.Args[1]

		// pass the domains to our functions
		getCSPApi(domain)
		getCSPHtml(domain)

		// dump the map into pretty json
		bytes, _ := json.MarshalIndent(cspObject, "", "    ")

		// print it to the user
		fmt.Println(string(bytes))
	} else {
		// if no domain or url is passed show the usage to the user.
		fmt.Println("[+] Usage: cspparse https://www.facebook.com")
	}
}

func getCSPApi(domain string) (string, error) {
	// set the parameters of the POST request
	// to url=<domain>
	client := &http.Client{Timeout: 2 * time.Second}

	// params := req.Param{
	// 	"url": domain,
	// }

	requestURL := fmt.Sprintf("https://csp-evaluator.withgoogle.com/getCSP?url=%s", url.QueryEscape(domain))

	// make the request
	// req.SetTimeout(2 * 1000 * 1000 * 1000)

	// r, err := req.Post("https://csp-evaluator.withgoogle.com/getCSP", params)
	resp, err := client.Get(requestURL)

	if err != nil {
		return "", errors.Wrap(err, "error making request")
	}

	// Defer closing the connection
	defer resp.Body.Close()

	var c cspStatus

	// JSON Decode Google's CSP response from JSON into the struct
	err = json.NewDecoder(resp.Body).Decode(&c)

	if err != nil {
		return "", errors.Wrap(err, "error decoding response JSON")
	}

	// If Google gave us status:"ok" there is a CSP.
	if c.Status == "ok" {
		// Rules are ';' delimited, split the CSP by ';' into rules
		cspResult := strings.Split(c.Csp, ";")

		for _, result := range cspResult {
			if result != "" {
				// Split the rule by the first space to get valid JSON
				rules := strings.Split(result, " ")

				// ex: default-src * data: blob:; -> "default-src": ["*","data:","blob:"],
				cspObject[rules[0]] = append(make([]string, 0), rules[1:]...)
			}
		}

	}

	return "", errors.New("no CSP for the domain given")
}
func getCSPHtml(domain string) string {
	// set the variable to the response body
	// from the function request()
	htmlCode := request(domain)
	// use goquery to parse the HTML.
	doc, err := goquery.NewDocumentFromReader(strings.NewReader((htmlCode)))
	if err != nil {
		log.Fatal(err)
	}
	// find all <meta> tags to see if there is a CSP.
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		// if we find a <meta> tag with the
		// attribute http-equiv set to "Content-Security-Policy"
		if name, _ := s.Attr("http-equiv"); name == "Content-Security-Policy" {
			// grab the CSP rule from the "content=" attribute
			description, _ := s.Attr("content")
			// delete spaces trailing after semicolons before splitting
			delSpace := strings.Replace(description, "; ", ";", -1)
			// split the CSP into rules by the ; separator
			cspResult := strings.Split(delSpace, ";")

			for _, result := range cspResult {
				if result != "" {
					// split the rule by the first space to get valid JSON
					rules := strings.Split(result, " ")
					// ex: default-src * data: blob:; -> "default-src": ["*","data:","blob:"],
					cspObject[rules[0]] = append(make([]string, 0), rules[1:]...)
				}
			}
		}
	})
	return ""
}
func request(url string) string {
	// make a GET request to the target domain / URL
	if url != "" {
		req.SetTimeout(2 * 1000 * 1000 * 1000)
		r, err := req.Get(url)

		if err != nil {
			fmt.Printf("Error making request:\n%s\n", err)
			os.Exit(1)
		}

		resp := r.Response()
		// close the connection
		defer resp.Body.Close()
		// read the response body from the target
		bodyBytes, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			fmt.Printf("Error: %s\n", err2)
		} else {
			bodyString := string(bodyBytes)
			// return the response body
			return bodyString
		}
	}
	return ""
}
