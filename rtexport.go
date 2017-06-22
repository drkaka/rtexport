package rtexport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"time"
)

// Record of usage information every five minutes
type Record struct {
	Begin        time.Time
	Spent        int
	Activity     string
	Category     string
	Productivity int8
}

type rtResponse struct {
	Notes      json.RawMessage `json:"notes"`
	RowHeaders json.RawMessage `json:"row_headers"`
	Rows       [][]interface{} `json:"rows"`
}

const reqURL = "https://www.rescuetime.com/anapi/data?pv=interval&rb=%s&re=%s&key=%s&format=json&rs=minute"

// GetRecords of a certain day
//
// day format: "YYYY-MM-DD"
func GetRecords(day, key string) ([]Record, error) {
	url := fmt.Sprintf(reqURL, day, day, key)
	resp, err := newCli().Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("bad request, %s", string(bodyBytes))
	}
	var records []Record
	var respMsg rtResponse
	if err := json.NewDecoder(resp.Body).Decode(&respMsg); err != nil {
		return nil, err
	}

	for _, one := range respMsg.Rows {
		if len(one) != 6 {
			return records, errors.New("record length wrong")
		}

		var rc Record

		// parse date
		beginT, ok := one[0].(string)
		if !ok {
			return records, fmt.Errorf("parsing time wrong: %v", one[0])
		}

		rc.Begin, err = time.Parse("2006-01-02T15:04:05", beginT)
		if err != nil {
			return records, err
		}

		// parse time spent
		spent, ok := one[1].(float64)
		if !ok {
			return records, fmt.Errorf("parsing duration wrong: %v", reflect.TypeOf(one[1]))
		}
		rc.Spent = int(spent)

		// parse activity
		rc.Activity, ok = one[3].(string)
		if !ok {
			return records, fmt.Errorf("parsing activity wrong: %v", reflect.TypeOf(one[3]))
		}

		// parse category
		rc.Category, ok = one[4].(string)
		if !ok {
			return records, fmt.Errorf("parsing category wrong: %v", reflect.TypeOf(one[4]))
		}

		// parse productivity
		productivity, ok := one[5].(float64)
		if !ok {
			return records, fmt.Errorf("parsing productivity wrong: %v", reflect.TypeOf(one[5]))
		}
		rc.Productivity = int8(productivity)

		records = append(records, rc)
	}
	return records, nil
}

func newCli() *http.Client {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	return &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}
}
