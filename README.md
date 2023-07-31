Certainly! Below is the `Readme.md` for your Golang library "translator":

# translator

[![Go Reference](https://pkg.go.dev/badge/github.com/lcapuano-app/translator.svg)](https://github.com/lcapuano-app/go-googletrans)

A Golang library that provides translation and language detection using the Google Translate API. It is a fork of the [`go-googletrans`](https://github.com/Conight/go-googletrans) library with added functionalities for language detection, default service URLs, and available languages.

## Installation

To use the `translator` library in your Go project, you need to install it using `go get`:

```bash
go get github.com/lcapuano-app/go-googletrans
```

## Usage

Here's a simple example of how to use the `translator` library:

```go
package main

import (
	"fmt"
	translator "github.com/lcapuano-app/go-googletrans"
)

func main() {
	// Create a new instance of the Translator.
	t := translator.New()

	// Translate text from English to Spanish.
	originText := "Hello, how are you?"
	sourceLanguage := "en"
	destinationLanguage := "es"

	translated, err := t.Translate(originText, sourceLanguage, destinationLanguage)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Source Language:", translated.Src)
	fmt.Println("Destination Language:", translated.Dest)
	fmt.Println("Original Text:", translated.Origin)
	fmt.Println("Translated Text:", translated.Text)

	// Detect the language of a given text.
	textToDetect := "¡Hola, cómo estás?"
	detected, err := t.DetectLanguage(textToDetect, destinationLanguage)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Detected Language:", detected.Src)
	fmt.Println("Confidence:", detected.Confidence)
}
```

## Configuration

You can also customize the Translator instance by providing configuration options using the `Config` struct. The available options are:

- `ServiceUrls`: A list of service URLs to be used for making API requests. If not provided, default service URLs will be used.
- `UserAgent`: A list of user agent strings used in the request headers. If not provided, a default user agent will be used.
- `Proxy`: The proxy URL to be used for making API requests. It should be in the format "http://proxy.example.com:8080". If not provided, no proxy will be used.

## Available Methods

The `translator` library provides the following methods:

- `Translate`: Translates text from one language to another using the Google Translate API.
- `DetectLanguage`: Detects the language of a given text using the Google Translate API.
- `GetValidLanguageKey`: Validates and returns the corresponding valid language code for a given language.
- `GetDefaultServiceUrls`: Returns the default service URLs used by the Translator.
- `GetAvailableLanguages`: Returns a map of available languages supported by the Google Translate API.

## Contribution

Contributions to the `translator` library are welcome! If you find any issues or have suggestions for improvement, feel free to open an issue or submit a pull request.

## License

This library is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---