// Package openfigi provides access to the OpenFIGI API.
// See https://openfigi.com/api for details.
package openfigi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gomodule/redigo/redis"
)

const apiURL = "https://api.openfigi.com/v1/mapping"

const defaultTimeout = 10 * time.Second

// ErrNotValidIdentifier is returned when the requested identifier type is invalid.
var ErrNotValidIdentifier = errors.New("not a valid identifier")

// ErrWrongStatus is returned whenever the openFIGI api returns a non 200 error.
var ErrWrongStatus = errors.New("wrong status received from api")

// ErrAPIError is returned when the openFIGI api results could not be parsed.
var ErrAPIError = errors.New("unknown api error occured")

// ErrNoIdentifierFound is returned when the requested identifier could not be found by the api.
var ErrNoIdentifierFound = errors.New("no identifier found")

// ErrCacheError is returned when we can't cache openFIGI api results in redis.
var ErrCacheError = errors.New("redis cache error")

// FIGI is the main OpenFIGI data.
type FIGI struct {
	FIGI                string `json:"figi"`
	SecurityType        string `json:"securityType"`
	MarketSector        string `json:"marketSector"`
	Ticker              string `json:"ticker"`
	Name                string `json:"name"`
	UniqueID            string `json:"uniqueID"`
	ExchangeCode        string `json:"exchCode"`
	ShareClassFIGI      string `json:"shareClassFIGI"`
	CompositeFIGI       string `json:"compositeFIGI"`
	SecurityType2       string `json:"securityType2"`
	SecurityDescription string `json:"securityDescription"`
	UniqueIDFutOpt      string `json:"uniqueIDFutOpt"`
}

// FIGIRequest is an OpenFIGI API Request.
type FIGIRequest struct {
	IDType       string `json:"idType,omitempty"`
	IDValue      string `json:"idValue,omitempty"`
	ExchangeCode string `json:"exchCode,omitempty"`
	MICCode      string `json:"micCode,omitempty"`
	Currency     string `json:"currency,omitempty"`
	MarketSector string `json:"marketSecDes,omitempty"`
	// private vars
	apiKey  string
	timeout time.Duration
}

// ValidIdentifiers is a list of valid openFIGI Request Identifiers.
var ValidIdentifiers = []string{
	"ID_ISIN",
	"ID_BB_UNIQUE",
	"ID_SEDOL",
	"ID_COMMON",
	"ID_WERTPAPIER",
	"ID_CUSIP",
	"ID_CINS",
	"ID_BB",
	"ID_ITALY",
	"ID_EXCH_SYMBOL",
	"ID_FULL_EXCHANGE_SYMBOL",
	"COMPOSITE_ID_BB_GLOBAL",
	"ID_BB_GLOBAL_SHARE_CLASS_LEVEL",
	"ID_BB_GLOBAL",
	"ID_BB_SEC_NUM_DES",
	"TICKER",
	"ID_CUSIP_8_CHR",
	"OCC_SYMBOL",
	"UNIQUE_ID_FUT_OPT",
	"OPRA_SYMBOL",
	"TRADING_SYSTEM_IDENTIFIER",
}

func isValidIdentifier(v string) bool {
	for _, i := range ValidIdentifiers {
		if v == i {
			return true
		}
	}
	return false
}

// NewRequest initializes a new request and checks if the idtype is valid.
func NewRequest(idtype, idvalue string) (*FIGIRequest, error) {
	if !isValidIdentifier(idtype) {
		return nil, ErrNotValidIdentifier
	}
	return &FIGIRequest{IDType: idtype, IDValue: idvalue, timeout: defaultTimeout}, nil
}

// APIKey sets the API key for openFIGI requests.
// Note that openFIGI works perfectly fine without API Key
// but requests are rate limited without one.
func (fr *FIGIRequest) APIKey(key string) {
	fr.apiKey = key
}

// Exchange sets the ExchangeCode for the openFIGI request.
func (fr *FIGIRequest) Exchange(exch string) {
	fr.ExchangeCode = exch
}

// Do performs the openFIGI request. Although openFIGI supports up
// to 5 queries per request, this implementation only supports 1.
// Errors returned are one of the package wide errors or generic
// http i/o or json parsing errors.
func (fr *FIGIRequest) Do() ([]*FIGI, error) {
	reqdata := []*FIGIRequest{fr}
	js, err := json.Marshal(reqdata)
	if err != nil {
		return nil, err
	}

	cached, err := getCache(js)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return cached, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(js))
	req.Header.Set("Content-Type", "application/json")
	if fr.apiKey != "" {
		req.Header.Set("X-OPENFIGI-APIKEY", fr.apiKey)
	}

	client := &http.Client{
		Timeout: fr.timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, ErrWrongStatus
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data := []map[string][]*FIGI{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		// No data was found, check if we maybe have an error.
		data := []map[string]string{}
		err = json.Unmarshal(body, &data)
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			return nil, ErrAPIError
		}
		e := data[0]["error"]
		if e == "No identifier found." {
			// This is the most common error.
			return nil, ErrNoIdentifierFound
		} else if e != "" {
			return nil, fmt.Errorf(data[0]["error"])
		}
		return nil, ErrAPIError
	}
	if len(data) == 0 {
		return nil, ErrAPIError
	}

	if err := setCache(js, data[0]["data"]); err != nil {
		return nil, ErrCacheError
	}
	return data[0]["data"], nil
}

var rpool *redis.Pool

// RedisCache sets up Redis to use as a cache for openFIGI data.
// If you don't call this on startup, no caching is performed.
func RedisCache(addr string) error {
	rpool = &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr)
		},
	}
	return nil
}

func getCache(key []byte) ([]*FIGI, error) {
	if rpool == nil {
		return nil, nil
	}
	c := rpool.Get()
	defer c.Close() // nolint: errcheck
	res, err := c.Do("GET", string(key))
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	b, err := redis.Bytes(res, err)
	if err != nil {
		return nil, err
	}
	data := []*FIGI{}
	err = json.Unmarshal(b, &data)
	return data, err
}

func setCache(key []byte, data []*FIGI) error {
	if rpool == nil {
		return nil
	}
	c := rpool.Get()
	defer c.Close() // nolint: errcheck

	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = c.Do("SET", string(key), string(js))
	return err
}
