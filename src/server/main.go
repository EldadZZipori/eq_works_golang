package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

const FILENAME = "data.json"
const TIMEFORMAT = "2006-01-02 15:04"
const LIMITER = 5

// Temporary holder for unsaved records
var record_tmp map[string]record = make(map[string]record)
var active_connections int = 0

type counters struct {
	sync.Mutex
	view  int
	click int
}

type record struct {
	Views  int64
	Clicks int64
}

func generateKey(cat string) string {
	curr_time := time.Now().Format(TIMEFORMAT)

	return cat + ":" + curr_time
}

// Dealing with increamenting a view value for a given key
// Creates a new entry if key does not exists
func viewInc(category string) {

	if val, check := record_tmp[generateKey(category)]; check {
		val.Views++
		record_tmp[generateKey(category)] = val
	} else {
		record_tmp[generateKey(category)] = record{Views: 1, Clicks: 0}
	}
}

// Note a redundent use of two funtion for increamenting. Should be integrated to one with an ENUM parameter to distinct between increaments
// Dealing with increamenting a view value for a given key
// Creates a new entry if key does not exists
func clickInc(category string) {

	if val, check := record_tmp[generateKey(category)]; check {
		val.Clicks++
		record_tmp[generateKey(category)] = val
	} else {
		record_tmp[generateKey(category)] = record{Views: 0, Clicks: 1}
	}
}

// Gets the whole records from the json file
func get_record() map[string]record {
	jsonFile, err := os.OpenFile(FILENAME, os.O_RDONLY|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]record)
		}
		fmt.Println(err)
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var records map[string]record

	if len(byteValue) > 0 {
		json.Unmarshal(byteValue, &records)
		return records
	}

	return make(map[string]record)
}

// Saves the current content of temp into the json file
// deletes the contents of tmp to avoid duplicacies
// save_record() should be updated to work with a db in order to avoid the redundency in loading the whole json file to memory
func save_record() bool {

	// No need to do anything if tmp is empty
	if len(record_tmp) == 0 {
		return true
	}

	var records map[string]record = get_record()

	for key, tmp_val := range record_tmp {
		if curr_val, check := records[key]; check {
			curr_val.Clicks = curr_val.Clicks + tmp_val.Clicks
			curr_val.Views = curr_val.Views + tmp_val.Views
			records[key] = curr_val

		} else {
			records[key] = record_tmp[key]
		}
		delete(record_tmp, key)
	}

	var jsonData []byte
	jsonData, err := json.MarshalIndent(records, "", "")
	if err != nil {
		return false
	}

	jsonFile, f_err := os.OpenFile(FILENAME, os.O_TRUNC|os.O_WRONLY, 0666)
	if f_err != nil {
		fmt.Println(f_err)
	}

	io.WriteString(jsonFile, string(jsonData))

	return true
}

var (
	// c is redundent as we have it in the record struct
	// c = counters{}

	content = []string{"sports", "entertainment", "business", "education"}
)

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to EQ Works 😎")
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	data := content[rand.Intn(len(content))]

	// Redundent
	// c.Lock()
	// c.view++
	// c.Unlock()

	viewInc(data)

	err := processRequest(r)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	// simulate random click call
	if rand.Intn(100) < 50 {
		processClick(data)
	}
}

func processRequest(r *http.Request) error {
	time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
	return nil
}

func processClick(data string) error {
	// Redundent
	// c.Lock()
	// c.click++
	// c.Unlock()

	clickInc(data)

	return nil
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if !isAllowed() {
		w.WriteHeader(429)
		return
	}
}

func isAllowed() bool {
	if active_connections < LIMITER {
		active_connections++
		return true
	}

	return false

}

func uploadCounters() error {
	return nil
}

func main() {
	http.HandleFunc("/", welcomeHandler)
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/stats/", statsHandler)

	record_save_ticker := time.NewTicker(5 * time.Second)
	active_connections_ticker := time.NewTicker(time.Second)

	// Saving an option to stop either procedure for future development if needed
	quit_record_save := make(chan bool)
	quit_rate_limit := make(chan bool)

	// Executing save_record as a go routine
	go func() {
		for {
			select {
			case <-record_save_ticker.C:
				save_record()
				fmt.Println("[*] Records saved")

			case <-quit_record_save:
				record_save_ticker.Stop()
				return
			}

		}
	}()

	go func() {
		for {
			select {
			case <-active_connections_ticker.C:
				if active_connections > 0 {
					active_connections--
				}
			case <-quit_rate_limit:
				record_save_ticker.Stop()
				return
			}
		}
	}()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
