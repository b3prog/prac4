package main

import (
	"fmt"
	"net/http"
	"net/url"
)

func isValidURL(rawURL string) bool {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return parsedURL.Scheme != "" && parsedURL.Host != ""
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			long := r.FormValue("url")

			if !isValidURL(long) {
				http.Error(w, "Invalid URL", http.StatusBadRequest)
				return
			}

			resp, err := http.PostForm("http://localhost:8081/set", url.Values{"url": {long}})
			if err != nil {
				http.Error(w, "Error communicating with storage service", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Fprintf(w, "Shortened URL: %s\n", resp.Header.Get("Location"))
			} else {
				http.Error(w, "Error creating shortened URL", resp.StatusCode)
			}
		} else if r.Method == http.MethodGet {
			resp, err := http.Get("http://localhost:8081/get" + r.URL.Path)
			if err != nil {
				http.Error(w, "Error communicating with storage service", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusTemporaryRedirect {
				http.Redirect(w, r, resp.Header.Get("Location"), http.StatusTemporaryRedirect)
			} else {
				http.NotFound(w, r)
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("URL shortener started on port 8080")
	http.ListenAndServe(":8080", nil)
}
