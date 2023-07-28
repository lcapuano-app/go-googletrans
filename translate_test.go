package translator

import (
	"testing"
)

// TestTranslator_Translate calls translate.translate.
func TestTranslator_Translate(t *testing.T) {
	origin := "你好，世界！"
	dest := "Hello World!"
	// c := Config{
	// 	Proxy:       "http://127.0.0.1:7890",
	// 	UserAgent:   []string{"Custom Agent"},
	// 	ServiceUrls: []string{"translate.google.com.hk"},
	// }
	trans := New()
	result, err := trans.Translate(origin, "auto", "en")

	if result.Text != dest || err != nil {
		t.Fatalf(`%q, %v, Want match for %q, nil`, result.Text, err, dest)
	}
}

func TestTranslator_DetectLanguage(t *testing.T) {
	dest := "en"
	origin := "Hello World!"
	trans := New()
	result, err := trans.DetectLanguage(origin, dest)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if result.Src != dest {
		t.Fatalf("%s should be %s", result.Src, dest)
	}

	origin = "hola mundo"
	result, err = trans.DetectLanguage(origin, "auto")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if result.Src != "es" {
		t.Fatalf("%s should be %s", result.Src, dest)
	}
	if result.Confidence < 0.5 {
		t.Fatalf("confidence %f should be over 0.5", result.Confidence)
	}
}

func TestTranslator_GetAvaliableLanguagesHTTP(t *testing.T) {
	trans := New()
	overwriteDefaultLanguages := false
	trans.GetAvaliableLanguagesHTTP(overwriteDefaultLanguages)
}
