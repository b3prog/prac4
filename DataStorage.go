package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
)

type URLStore struct {
	urls map[string]string
	sync.RWMutex
}

type Statistic struct {
	SourceIP  string `json:"source_ip"`
	URL       string `json:"url"`
	Short     string `json:"short"`
	Timestamp string `json:"timestamp"`
}

func (s *URLStore) Get(short string) (string, bool) {
	s.RLock()
	defer s.RUnlock()
	long, ok := s.urls[short]
	return long, ok
}

func (s *URLStore) Set(short, long string) error {
	s.Lock()
	defer s.Unlock()

	for existingShort, existingLong := range s.urls {
		if existingLong == long {
			return fmt.Errorf("URL already exists with the given long URL: %s", existingShort)
		}
	}

	s.urls[short] = long
	return nil
}

func (s *URLStore) Save(filename string) error {
	s.RLock()
	defer s.RUnlock()

	var lines []string
	for short, long := range s.urls {
		lines = append(lines, short+" "+long)
	}

	content := strings.Join(lines, "\n")
	return os.WriteFile(filename, []byte(content), 0644)
}

func (s *URLStore) Load(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			if err := s.Save(filename); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			parts := strings.Split(line, " ")
			s.Set(parts[0], parts[1])
		}
	}

	return nil
}

func saveStatsToFile(stats []Statistic, filename string) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

func main() {
	store := &URLStore{urls: make(map[string]string)}
	stats := make([]Statistic, 0)
	statsFilename := "stats.json"

	err := store.Load("urls.txt")
	if err != nil {
		fmt.Println("Error loading URLs:", err)
		os.Exit(1)
	}

	http.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			long := r.FormValue("url")
			short := fmt.Sprintf("short.ly/%d", len(store.urls)+1)
			err := store.Set(short, long)
			if err == nil {
				store.Save("urls.txt")
				w.Header().Set("Location", short)
				w.WriteHeader(http.StatusOK)
			} else {
				http.Error(w, "Error saving URL", http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/get/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			short := r.URL.Path[len("/get/"):]
			long, ok := store.Get(short)
			if ok {
				fmt.Fprintf(w, "%s", long) 
			} else {
				http.NotFound(w, r)
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var stat Statistic
			err := json.NewDecoder(r.Body).Decode(&stat)
			if err != nil {
				http.Error(w, "Error decoding JSON", http.StatusBadRequest)
				return
			}

			stats = append(stats, stat)
			err = saveStatsToFile(stats, statsFilename)
			if err != nil {
				http.Error(w, "Error saving stats", http.StatusInternalServerError)
				return
			}

			fmt.Printf("Received data from statistics service: %+v\n", stat)
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Data storage started on port 8081")
	http.ListenAndServe(":8081", nil)
}
