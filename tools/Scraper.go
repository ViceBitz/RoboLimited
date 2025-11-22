package tools

import (
    "net/http"
	"fmt"
	"io"
)
/*
Scrapes HTML source code off webpages.
*/

//Retrieves page source of a URL
func GetPageSource(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch url: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}
	html := string(body)
	return html, nil
}


