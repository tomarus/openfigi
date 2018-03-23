package openfigi

import "testing"

func TestRequest(t *testing.T) {
	// Enable this to enable Redis caching.
	// RedisCache("192.168.0.3:6379")

	req, err := NewRequest("ID_ISIN", "US3623931009")
	if err != nil {
		t.Fatal(err)
	}

	req.Exchange("US")

	res, err := req.Do()
	if err != nil {
		t.Fatal(err)
	}

	if res[0].Ticker != "GTT" {
		t.Fatalf("Expected ticker=GTT, but is %v", res[0].Ticker)
	}
	if res[0].Name != "GTT COMMUNICATIONS INC" {
		t.Fatalf("Expected name=GTT COMMUNICATIONS INC, but is %v", res[0].Name)
	}
}
