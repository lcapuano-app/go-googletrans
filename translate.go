package translator

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config basic config.
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

func randomChoose(slice []string) string {
	return slice[rand.Intn(len(slice))]
}

type addHeaderTransport struct {
	T              http.RoundTripper
	defaultHeaders map[string]string
}

func (adt *addHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range adt.defaultHeaders {
		req.Header.Add(k, v)
	}
	return adt.T.RoundTrip(req)
}

func newAddHeaderTransport(T http.RoundTripper, defaultHeaders map[string]string) *addHeaderTransport {
	if T == nil {
		T = http.DefaultTransport
	}
	return &addHeaderTransport{T, defaultHeaders}
}

func New(config ...Config) *Translator {
	rand.Seed(time.Now().Unix())
	var c Config
	if len(config) > 0 {
		c = config[0]
	}
	// set default value
	if len(c.ServiceUrls) == 0 {
		c.ServiceUrls = []string{"translate.google.com"}
	}
	if len(c.UserAgent) == 0 {
		c.UserAgent = []string{defaultUserAgent}
	}

	host := randomChoose(c.ServiceUrls)
	userAgent := randomChoose(c.UserAgent)
	proxy := c.Proxy

	transport := &http.Transport{}
	// Skip verifies the server's certificate chain and host name.
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // skip verify
	// set proxy
	if strings.HasPrefix(proxy, "http") {
		proxyUrl, _ := url.Parse(proxy)
		transport.Proxy = http.ProxyURL(proxyUrl) // set proxy
	}

	// new client with custom headers
	client := &http.Client{
		Transport: newAddHeaderTransport(transport, map[string]string{
			"User-Agent": userAgent,
		}),
	}

	ta := Token(host, client)
	return &Translator{
		host:   host,
		client: client,
		ta:     ta,
	}
}

// Translate given content.
// Set src to `auto` and system will attempt to identify the source language automatically.
func (a *Translator) Translate(origin, src, dest string) (*Translated, error) {
	// check src & dest
	src = strings.ToLower(src)
	dest = strings.ToLower(dest)
	text, err := a.translate(a.client, origin, src, dest)
	if err != nil {
		return nil, err
	}
	result := &Translated{
		Src:    src,
		Dest:   dest,
		Origin: origin,
		Text:   text,
	}
	return result, nil
}

func (a *Translator) translate(client *http.Client, origin, src, dest string) (string, error) {
	req, err := a.getReq(client, origin, src, dest)
	if err != nil {
		return "", err
	}
	// do request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		var sentences sentences
		err = json.Unmarshal(body, &sentences)
		if err != nil {
			return "", err
		}

		translated := ""
		// parse trans
		for _, s := range sentences.Sentences {
			translated += s.Trans
		}
		return translated, nil
	} else {
		return "", fmt.Errorf("expected statusCode 200, got: %d; resp: %+v", resp.StatusCode, resp)
	}
}

func buildParams(query, src, dest, token string) map[string]string {
	params := map[string]string{
		"client": "gtx",
		"sl":     src,
		"tl":     dest,
		"hl":     dest,
		"tk":     token,
		"q":      query,
	}
	return params
}

// Checks if the requested language exists on languages const.
// It acepts short lang key (en, es, pt, etc..) and full language name (english, spanish, portuguese, etc..)
//
// Returns key, nil || "auto", error
func (a *Translator) GetValidLanguageKey(lang string) (string, error) {
	lang = strings.ToLower(lang)
	for key, val := range languages {
		if key == lang || val == lang {
			return key, nil
		}
	}
	return defaultLanguage, fmt.Errorf("invalid language '%s'", lang)
}

func (a *Translator) GetAvaliableLanguages() map[string]string {
	return languages
}

// Detects the provided text writen language. Pass dest as "auto" to identify the source language automatically.
func (a *Translator) DetectLanguage(origin, dest string) (LDResponse, error) {
	dest = strings.ToLower(dest)
	detected, err := a.detect(a.client, origin, dest)
	if err != nil {
		return detected, err
	}
	return detected, nil
}

func (a *Translator) detect(client *http.Client, origin, dest string) (LDResponse, error) {
	var detected LDResponse
	req, err := a.getReq(client, origin, "auto", dest)
	if err != nil {
		return detected, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return detected, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return detected, fmt.Errorf("expected statusCode 200, got: %d; resp: %+v", resp.StatusCode, resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return detected, err
	}

	err = json.Unmarshal(body, &detected)
	if err != nil {
		return detected, err
	}
	translated := ""
	for _, s := range detected.Sentences {
		translated += s.Trans
	}

	return detected, nil
}

func (a *Translator) getReq(client *http.Client, origin, src, dest string) (*http.Request, error) {
	tk, err := a.ta.do(origin)
	if err != nil {
		return nil, err
	}

	tranUrl := fmt.Sprintf("https://%s/translate_a/single", a.host)
	req, err := http.NewRequest("GET", tranUrl, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	// params from chrome translate extension
	params := buildParams(origin, src, dest, tk)
	for i := range params {
		q.Add(i, params[i])
	}
	q.Add("dt", "t")
	q.Add("dt", "bd")
	q.Add("dj", "1")
	q.Add("source", "popup")
	req.URL.RawQuery = q.Encode()

	return req, nil
}

func GetDefaultServiceUrls() []string {
	return defaultServiceUrls
}

// Gets all avaliable languages from https://cloud.google.com/translate/docs/languages
//
// Use overwriteDefaultLanguages if you encounter problems with the requested language
func (a *Translator) GetAvaliableLanguagesHTTP(overwriteDefaultLanguages bool) (map[string]string, error) {
	httpLangs := languages
	url := "https://cloud.google.com/translate/docs/languages"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return httpLangs, err
	}
	req.Header.Add("Cookie", "_ga_devsite=GA1.3.3578724760.1690567683")

	res, err := client.Do(req)
	if err != nil {
		return httpLangs, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return httpLangs, err
	}

	bodyStr := string(body)
	ini, end := strings.Index(bodyStr, "<table>"), strings.Index(bodyStr, "</table>")
	table := bodyStr[ini:end]
	ini, end = strings.Index(table, "<tbody>"), strings.Index(table, "</tbody>")
	tbody := table[ini:end]
	tbody = strings.ReplaceAll(tbody, "<tbody>", "")
	tbody = strings.ReplaceAll(tbody, "</tbody>", "")
	trs := strings.Split(tbody, "<tr>")

	for _, tr := range trs {
		ini, end = strings.Index(tr, "<td>"), strings.Index(tr, "</td>")
		if ini < 0 || end < 0 {
			continue
		}
		fullname := tr[(ini + 4):end]
		if len(fullname) == 0 {
			continue
		}
		end = strings.Index(tr, "</code>")
		ini = end - 2
		shortname := tr[ini:end]
		if len(shortname) == 0 {
			continue
		}
		shortname = strings.ToLower(shortname)
		fullname = strings.ToLower(fullname)
		httpLangs[shortname] = fullname
	}

	if overwriteDefaultLanguages {
		languages = httpLangs
	}
	return httpLangs, nil
}
