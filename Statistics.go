package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type Statistic struct {
	SourceIP  string    `json:"source_ip"`
	URL       string    `json:"url"`
	Short     string    `json:"short_url"`
	Timestamp time.Time `json:"timestamp"`
}

type ReportEntry struct {
	ID           int    `json:"Id"`
	PID          *int   `json:"Pid"`
	OriginalURL  string `json:"FullURL,omitempty"`
	ShortURL     string `json:"ShortenURL,omitempty"`
	SourceIP     string `json:"SourceIP"`
	TimeInterval string `json:"TimeInterval"`
	Count        int    `json:"Count"`
}

type ReportData struct {
	Entries []ReportEntry `json:"entries"`
}

type DetailReport struct {
	Count   int                      `json:"Count,omitempty"`
	Details map[string]*DetailReport `json:"Details,omitempty"`
}

var reportData = ReportData{
	Entries: []ReportEntry{},
}

var mu sync.Mutex

func (dr *DetailReport) getOrCreateDetail(key string) *DetailReport {
	if dr.Details == nil {
		dr.Details = make(map[string]*DetailReport)
	}
	if _, ok := dr.Details[key]; !ok {
		dr.Details[key] = &DetailReport{Count: 0}
	}
	return dr.Details[key]
}

func generateReport(detailsOrder []string) DetailReport {
	mu.Lock()
	defer mu.Unlock()

	report := DetailReport{Count: 0}

	for _, entry := range reportData.Entries {
		currLevel := &report
		currLevel.Count += entry.Count

		for _, level := range detailsOrder {
			switch level {
			case "SourceIP":
				currLevel = currLevel.getOrCreateDetail(entry.SourceIP)
			case "TimeInterval":
				currLevel = currLevel.getOrCreateDetail(entry.TimeInterval)
			case "URL":
				currLevel = currLevel.getOrCreateDetail(fmt.Sprintf("%s (%s)", entry.OriginalURL, entry.ShortURL))
			}

			currLevel.Count += entry.Count
		}
	}

	return report
}

func addDataToReport(stat Statistic) {
	mu.Lock()
	defer mu.Unlock()



	intervalStart := stat.Timestamp.Truncate(time.Minute)
	intervalEnd := intervalStart.Add(time.Minute)

	entry := ReportEntry{
		OriginalURL:  stat.URL,
		ShortURL:     stat.Short,
		SourceIP:     stat.SourceIP,
		TimeInterval: fmt.Sprintf("%s - %s", intervalStart.Format("15:04"), intervalEnd.Format("15:04")),
		Count:        1,
	}

	reportData.Entries = append(reportData.Entries, entry)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			sourceIP := r.RemoteAddr
			shortURL := r.PostFormValue("url")
			longURL := r.PostFormValue("long_url")
			timestamp, _ := time.Parse(time.RFC3339, r.PostFormValue("timestamp"))

			stat := Statistic{
				SourceIP:  sourceIP,
				URL:       longURL,
				Short:     shortURL,
				Timestamp: timestamp,
			}

			fmt.Printf("Received data from URL shortener: %+v\n", stat)

			data, _ := json.Marshal(stat)
			resp, err := http.Post("http://localhost:8081", "application/json", bytes.NewBuffer(data))
			if err != nil {
				http.Error(w, "Error sending data to storage service", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			addDataToReport(stat)

			report := generateReport([]string{"URL", "SourceIP", "TimeInterval"})


			reportData, _ := json.MarshalIndent(report, "", "\t")
			err = ioutil.WriteFile("report.json", reportData, 0644)
			if err != nil {
				http.Error(w, "Error saving report to file", http.StatusInternalServerError)
				return
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	fmt.Println("Statistics service started on port 8082")
	http.ListenAndServe(":8082", nil)
}
