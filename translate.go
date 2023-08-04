package translator

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config basic opts.
type Config struct {
	ServiceUrls []string
	UserAgent   []string
	Proxy       string
}

// Translated result object.
type Translated struct {
	Src    string // source language
	Dest   string // destination language
	Origin string // original text
	Text   string // translated text
}

type sentences struct {
	Sentences []sentence `json:"sentences"`
}

type sentence struct {
	Trans   string `json:"trans"`
	Orig    string `json:"orig"`
	Backend int    `json:"backend"`
}

// Language detection (LD) response
type LDResponse struct {
	Sentences  []sentence  `json:"sentences"`
	Src        string      `json:"src,omitempty"`
	Spell      interface{} `json:"spell,omitempty"`
	Confidence float64     `json:"confidence,omitempty"`
	LdResult   LDResult    `json:"ld_result,omitempty"`
}

// Language detection (LD) result
type LDResult struct {
	Srclangs            []string  `json:"srclangs,omitempty"`
	SrclangsConfidences []float64 `json:"srclangs_confidences,omitempty"`
	ExtendedSrclangs    []string  `json:"extended_srclangs,omitempty"`
}

type Translator struct {
	host   string
	client *http.Client
	ta     *tokenAcquirer
}

type addHeaderTransport struct {
	T              http.RoundTripper
	defaultHeaders map[string]string
}

func randomChoose(slice []string) string {
	return slice[rand.Intn(len(slice))]
}

// New is a function that creates a new instance of the Translator with the specified configuration.
// If no configuration is provided, default values are used.
//
// Parameters:
// - config: Variadic parameter that accepts optional configurations for the Translator.
//   - ServiceUrls: A slice of service URLs used for making API requests. If not provided, defaultServiceUrls will be used.
//   - UserAgent: A slice of user agent strings used in the request headers. If not provided, defaultUserAgent will be used.
//   - Proxy: The proxy URL to be used for making API requests. It should be in the format "http://proxy.example.com:8080".
//     If not provided, no proxy will be used.
//
// Returns:
// - *Translator: A new instance of the Translator with the specified configurations.
//
// Example Usage:
//
//	config := Config{
//	  ServiceUrls: []string{"https://api.example.com/translate", "https://api.example.org/translate"},
//	  UserAgent:   []string{"MyApp/1.0", "MyOtherApp/2.0"},
//	  Proxy:       "http://proxy.example.com:8080",
//	}
//	translator := New(config)
func New(config ...Config) *Translator {
	rand.Seed(time.Now().Unix())
	var c Config

	// Set default values if not provided in the configuration.
	if len(c.ServiceUrls) == 0 {
		c.ServiceUrls = defaultServiceUrls
	}
	if len(c.UserAgent) == 0 {
		c.UserAgent = []string{defaultUserAgent}
	}

	// Randomly choose a service URL and user agent from the provided configurations.
	host := randomChoose(c.ServiceUrls)
	userAgent := randomChoose(c.UserAgent)
	proxy := c.Proxy

	// Create an HTTP transport with custom settings, including skipping certificate verification and setting a proxy if provided.
	transport := &http.Transport{}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	if strings.HasPrefix(proxy, "http") {
		proxyUrl, _ := url.Parse(proxy)
		transport.Proxy = http.ProxyURL(proxyUrl)
	}

	// Create an HTTP client with custom headers, including the selected user agent.
	client := &http.Client{
		Transport: newAddHeaderTransport(transport, map[string]string{
			"User-Agent": userAgent,
		}),
	}

	// Initialize the token service (ta) using the selected host and client.
	ta := Token(host, client)

	// Create and return a new instance of the Translator with the selected host, client, and token service.
	return &Translator{
		host:   host,
		client: client,
		ta:     ta,
	}
}

// RoundTrip is a method of the addHeaderTransport struct that adds default headers to an outgoing HTTP request and executes the request using the underlying RoundTripper (T).
// It modifies the request by adding the default headers specified when creating the addHeaderTransport.
//
// Parameters:
// - req: The HTTP request to be modified and sent.
//
// Returns:
// - *http.Response: The HTTP response received as a result of the request.
// - error: An error if there is any issue with the request or response.
//
// Example Usage:
//
//	defaultHeaders := map[string]string{
//	  "Authorization": "Bearer YOUR_ACCESS_TOKEN",
//	  "User-Agent":    "MyApp/1.0",
//	}
//	customTransport := newAddHeaderTransport(nil, defaultHeaders)
//	client := &http.Client{Transport: customTransport}
//	req, _ := http.NewRequest("GET", "https://example.com/api/resource", nil)
//	resp, err := customTransport.RoundTrip(req)
//	if err != nil {
//	  fmt.Println("Error:", err)
//	  return
//	}
//	defer resp.Body.Close()
//	// Process the HTTP response.
func (adt *addHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add the default headers to the outgoing HTTP request.
	for k, v := range adt.defaultHeaders {
		req.Header.Add(k, v)
	}

	// Execute the request using the underlying RoundTripper (T).
	return adt.T.RoundTrip(req)
}

// newAddHeaderTransport is a function that creates a new HTTP transport with added default headers.
// It takes an existing HTTP RoundTripper (T) and a map of defaultHeaders to be added to the transport.
//
// Parameters:
//   - T: The existing HTTP RoundTripper to be used for the transport. If T is nil, http.DefaultTransport will be used.
//   - defaultHeaders: A map of default headers to be added to the transport. The keys represent the header names,
//     and the values represent the header values.
//
// Returns:
// - *addHeaderTransport: A new HTTP transport that includes the provided defaultHeaders in the request headers.
//
// Example Usage:
//
//	defaultHeaders := map[string]string{
//	  "Authorization": "Bearer YOUR_ACCESS_TOKEN",
//	  "User-Agent":    "MyApp/1.0",
//	}
//	customTransport := newAddHeaderTransport(nil, defaultHeaders)
//	client := &http.Client{Transport: customTransport}
//	// Use the 'client' to make HTTP requests with the added default headers.
func newAddHeaderTransport(T http.RoundTripper, defaultHeaders map[string]string) *addHeaderTransport {
	// If the provided RoundTripper is nil, use http.DefaultTransport.
	if T == nil {
		T = http.DefaultTransport
	}

	// Return a new addHeaderTransport with the provided RoundTripper and defaultHeaders.
	return &addHeaderTransport{T, defaultHeaders}
}

// Translate is a public method of the Translator struct that allows users to translate text from one language to another using the Google Translate API.
//
// Parameters:
// - origin: The text to be translated.
// - src: The language code of the source text. (e.g., "en" for English, "es" for Spanish)
// - dest: The language code for the desired translation output. (e.g., "es" for Spanish, "fr" for French)
//
// Returns:
// - *Translated: A struct containing the translation result, including the source language, destination language, original text, and translated text.
// - error: An error if there is any issue with the translation or HTTP request.
//
// Example Usage:
//
//	translator := NewTranslator("YOUR_API_KEY")
//	originText := "Hello, how are you?"
//	sourceLanguage := "en"
//	destinationLanguage := "es"
//	translated, err := translator.Translate(originText, sourceLanguage, destinationLanguage)
//	if err != nil {
//	  fmt.Println("Error:", err)
//	  return
//	}
//	fmt.Println("Source Language:", translated.Src)
//	fmt.Println("Destination Language:", translated.Dest)
//	fmt.Println("Original Text:", translated.Origin)
//	fmt.Println("Translated Text:", translated.Text)
func (a *Translator) Translate(origin, src, dest string) (*Translated, error) {
	// Convert the source and destination language codes to lowercase for consistency.
	src = strings.ToLower(src)
	dest = strings.ToLower(dest)

	// Perform the translation using the internal translate method.
	text, err := a.translate(a.client, origin, src, dest)
	if err != nil {
		return nil, err
	}

	// Create a new Translated struct to store the translation result.
	result := &Translated{
		Src:    src,    // Source language code.
		Dest:   dest,   // Destination language code.
		Origin: origin, // Original text.
		Text:   text,   // Translated text.
	}

	// Return the Translated struct containing the translation result.
	return result, nil
}

// translate is a private method of the Translator struct that performs the translation using the Google Translate API.
//
// Parameters:
// - client: The *http.Client to be used for making the HTTP request to the Google Translate API.
// - origin: The text to be translated.
// - src: The language code of the source text. (e.g., "en" for English, "es" for Spanish)
// - dest: The language code for the desired translation output. (e.g., "es" for Spanish, "fr" for French)
//
// Returns:
// - string: The translated text as a result of the translation.
// - error: An error if there is any issue with the translation or HTTP request.
//
// Example Usage:
//
//	translator := New()
//	client := &http.Client{}
//	originText := "Hello, how are you?"
//	sourceLanguage := "en"
//	destinationLanguage := "es"
//	translatedText, err := translator.translate(client, originText, sourceLanguage, destinationLanguage)
//	if err != nil {
//	  fmt.Println("Error:", err)
//	  return
//	}
//	fmt.Println("Translated Text:", translatedText)
func (a *Translator) translate(client *http.Client, origin, src, dest string) (string, error) {
	// Get the HTTP request for the API call.
	req, err := a.getReq(client, origin, src, dest)
	if err != nil {
		return "", err
	}

	// Perform the HTTP request to the Google Translate API.
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check if the API response has a status code of 200 (OK).
	if resp.StatusCode == 200 {
		// Read the response body.
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		// Unmarshal the JSON response into the 'sentences' variable.
		var sentences sentences
		err = json.Unmarshal(body, &sentences)
		if err != nil {
			return "", err
		}

		// Combine all translated sentences into a single string.
		translated := ""
		for _, s := range sentences.Sentences {
			translated += s.Trans
		}

		// Return the translated text.
		return translated, nil
	} else {
		return "", fmt.Errorf("expected statusCode 200, got: %d; resp: %+v", resp.StatusCode, resp)
	}
}

// buildParams is a helper function used to construct the query parameters for making a Google Translate API call.
//
// Parameters:
// - query: The text to be translated.
// - src: The language code of the source text. (e.g., "en" for English, "es" for Spanish)
// - dest: The language code for the desired translation output. (e.g., "es" for Spanish, "fr" for French)
// - token: The translation token used for API authentication and request validation.
//
// Returns:
// - map[string]string: A map of query parameters with keys representing the parameter names and values representing their corresponding values.
//
// Example Usage:
//
//	queryText := "Hello, how are you?"
//	sourceLanguage := "en"
//	destinationLanguage := "es"
//	authToken := "YOUR_AUTH_TOKEN"
//	params := buildParams(queryText, sourceLanguage, destinationLanguage, authToken)
//	// Use the 'params' map to construct the API call query string.
func buildParams(query, src, dest, token string) map[string]string {
	params := map[string]string{
		"client": "gtx", // The translation client identifier.
		"sl":     src,   // The source language code.
		"tl":     dest,  // The target (destination) language code.
		"hl":     dest,  // The language code for the target language in the output (used for transliteration).
		"tk":     token, // The translation token for API authentication.
		"q":      query, // The text to be translated.
	}
	return params
}

// GetValidLanguageKey is a method of the Translator struct that validates and returns the corresponding valid language key for a given language.
// It checks whether the provided language is a valid language code or language name in the 'languages' map.
//
// Parameters:
// - lang: The language code or language name to be validated.
//
// Returns:
// - string: The valid language code (key) corresponding to the provided language.
// - error: An error if the provided language is not a valid language code or language name in the 'languages' map.
//
// Example Usage:
//
//	language := "es"
//	validLang, err := translator.GetValidLanguageKey(language)
//	if err != nil {
//	  fmt.Println("Error:", err)
//	  return
//	}
//	fmt.Println("Valid Language Code:", validLang)
func GetValidLanguageKey(lang string) (string, error) {
	// Convert the provided language to lowercase for consistency.
	lang = strings.ToLower(lang)

	// Check if the provided language exists as a key or value in the 'languages' map.
	for key, val := range languages {
		if key == lang || val == lang {
			return key, nil
		}
	}

	// If the provided language is not valid, return the default language key and an error.
	return defaultLanguage, fmt.Errorf("invalid language '%s'", lang)
}

// DetectLanguage is a public method of the Translator struct that allows users to detect the language of a given text and obtain its translation.
//
// Parameters:
// - origin: The language code or "auto" to automatically detect the language of the input text.
// - dest: The language code for the desired translation output. (e.g., "es" for Spanish, "fr" for French)
//
// Returns:
// - LDResponse: The detected language and translated text as a result of the language detection.
// - error: An error if there is any issue with the language detection or HTTP request.
//
// Example Usage:
//
//	translator := New()
//	originText := "Hello, how are you?"
//	destinationLanguage := "es"
//	detected, err := translator.DetectLanguage("auto", destinationLanguage)
//	if err != nil {
//	  fmt.Println("Error:", err)
//	  return
//	}
//	fmt.Println("Detected Language:", detected.Lang)
//	fmt.Println("Translated Text:", detected.Trans)
func (a *Translator) DetectLanguage(origin, dest string) (LDResponse, error) {
	// Convert the destination language to lowercase for consistency.
	dest = strings.ToLower(dest)

	// Call the internal detect method to perform language detection and translation.
	detected, err := a.detect(a.client, origin, dest)
	if err != nil {
		return detected, err
	}

	// Return the detected language and translated text.
	return detected, nil
}

// detect is a method of the Translator struct that performs language detection using the Google Translate API.
// It sends a request to the API with the provided origin and destination languages and returns the detected language and translated text.
//
// Parameters:
// - client: The *http.Client to be used for making the HTTP request to the Google Translate API.
// - origin: The language code or "auto" to automatically detect the language of the input text.
// - dest: The language code for the desired translation output.
//
// Returns:
// - LDResponse: The detected language and translated text as a result of the language detection.
// - error: An error if there is any issue with the HTTP request or JSON unmarshaling.
//
// Example Usage:
//
//	translator := New()
//	client := &http.Client{}
//	originText := "Hello, how are you?"
//	destinationLanguage := "es"
//	detected, err := translator.detect(client, "auto", destinationLanguage)
//	if err != nil {
//	  fmt.Println("Error:", err)
//	  return
//	}
//	fmt.Println("Detected Language:", detected.Lang)
//	fmt.Println("Translated Text:", detected.Trans)
func (a *Translator) detect(client *http.Client, origin, dest string) (LDResponse, error) {
	// Initialize an empty LDResponse to store the detected language and translated text.
	var detected LDResponse

	// Create an HTTP request with the provided origin and destination languages.
	req, err := a.getReq(client, origin, "auto", dest)
	if err != nil {
		return detected, err
	}

	// Send the HTTP request to the Google Translate API.
	resp, err := client.Do(req)
	if err != nil {
		return detected, err
	}
	defer resp.Body.Close()

	// Check if the API response has a status code of 200 (OK).
	if resp.StatusCode != 200 {
		return detected, fmt.Errorf("expected statusCode 200, got: %d; resp: %+v", resp.StatusCode, resp)
	}

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return detected, err
	}

	// Unmarshal the JSON response into the detected variable.
	err = json.Unmarshal(body, &detected)
	if err != nil {
		return detected, err
	}

	// Combine all translated sentences into a single string.
	translated := ""
	for _, s := range detected.Sentences {
		translated += s.Trans
	}

	// Return the detected language and translated text.
	return detected, nil
}

// getReq is a method of the Translator struct that constructs an HTTP GET request to make a Google Translate API call.
// The API call aims to translate text from the source language to the destination language using a given translation token (tk).
//
// Parameters:
// - client: The *http.Client to be used for making the HTTP request to the Google Translate API.
// - origin: The original text that needs to be translated.
// - src: The language code of the source text. (e.g., "en" for English, "es" for Spanish)
// - dest: The language code for the desired translation output. (e.g., "es" for Spanish, "fr" for French)
//
// Returns:
// - *http.Request: The constructed HTTP request with all the necessary parameters to make the Google Translate API call.
// - error: An error if there is any issue with the API call or token retrieval.
//
// Example Usage:
//
//	translator := New()
//	client := &http.Client{}
//	originText := "Hello, how are you?"
//	sourceLanguage := "en"
//	destinationLanguage := "es"
//	req, err := translator.getReq(client, originText, sourceLanguage, destinationLanguage)
//	if err != nil {
//	  fmt.Println("Error:", err)
//	  return
//	}
//	// Use the 'req' object to execute the API call.
func (a *Translator) getReq(client *http.Client, origin, src, dest string) (*http.Request, error) {
	// Get the translation token (tk) for API authentication.
	tk, err := a.ta.do(origin)
	if err != nil {
		return nil, err
	}

	// Build the URL for the API call.
	tranUrl := fmt.Sprintf("https://%s/translate_a/single", a.host)
	req, err := http.NewRequest("GET", tranUrl, nil)
	if err != nil {
		return nil, err
	}

	// Construct the query parameters for the API call.
	q := req.URL.Query()
	params := buildParams(origin, src, dest, tk)
	for i := range params {
		q.Add(i, params[i])
	}

	// Add additional parameters to the query string.
	q.Add("dt", "t")         // Include translations in the response.
	q.Add("dt", "bd")        // Include dictionary and alternate translations in the response.
	q.Add("dj", "1")         // Include JSON format in the response.
	q.Add("source", "popup") // Identify the source of the translation as "popup".

	// Encode the query parameters and set them in the request URL.
	req.URL.RawQuery = q.Encode()

	return req, nil
}

func GetDefaultServiceUrls() []string {
	return defaultServiceUrls
}

func (a *Translator) GetAvaliableLanguages() map[string]string {
	return languages
}
