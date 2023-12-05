package main

import (
  "fmt"
  "io/ioutil"
  "net/http"
  "net/url"
  "time"
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
        shortURL := resp.Header.Get("Location")
        fmt.Fprintf(w, "Shortened URL: %s", shortURL)

      } else if resp.StatusCode == http.StatusConflict {
        fmt.Fprintf(w, "Shortened URL already exists")
      } else {
        http.Error(w, "Error creating shortened URL", resp.StatusCode)
      }
    } else if r.Method == http.MethodGet {
      shortURL := r.URL.Path[1:]
      if shortURL != "" {
        // Get user's IP address
        userIP := r.RemoteAddr

        // Get current timestamp
        timestamp := time.Now().Format(time.RFC3339)

        // Get long URL
        resp, err := http.Get("http://localhost:8081/get/" + shortURL)
        if err != nil {
          http.Error(w, "Error getting long URL", http.StatusInternalServerError)
          return
        }
        defer resp.Body.Close()

        longURLBytes, err := ioutil.ReadAll(resp.Body)
        if err != nil {
          http.Error(w, "Error reading long URL", http.StatusInternalServerError)
          return
        }

        longURL := string(longURLBytes)

        // Send data to statistics service
        _, err = http.PostForm("http://localhost:8082", url.Values{"ip": {userIP}, "url": {shortURL}, "long_url": {longURL}, "timestamp": {timestamp}})
        if err != nil {
          http.Error(w, "Error sending data to statistics service", http.StatusInternalServerError)
          return
        }

        fmt.Printf("Sent data to statistics service: {SourceIP:%s URL:%s Short:%s Timestamp:%s}\n", userIP, longURL, shortURL, timestamp)

        http.Redirect(w, r, longURL, http.StatusTemporaryRedirect)
        return
      }
      http.NotFound(w, r)
    } else {
      w.WriteHeader(http.StatusMethodNotAllowed)
    }
  })

  fmt.Println("URL Shortener started on port 8080")
  http.ListenAndServe(":8080", nil)
}
