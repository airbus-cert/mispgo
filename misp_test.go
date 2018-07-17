package misp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

var (
	mux    *http.ServeMux
	client *Client
	server *httptest.Server
)

func setup() {
	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	// client configured to use test server
	client = &Client{}
	client.BaseURL, _ = url.Parse(server.URL)
	client.APIKey = "dummyapikeyfortests"
}

// shamely stolen from go-github/github/github_test.go
func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

// shamely stolen from go-github/github/github_test.go
func testHeader(t *testing.T, r *http.Request, header, want string) {
	if got := r.Header.Get(header); got != want {
		t.Errorf("Header.Get(%q) returned %q, want %q", header, got, want)
	}
}

func testAuthentication(t *testing.T, r *http.Request) {
	testHeader(t, r, "Authorization", client.APIKey)
}

func Test_SearchAttribute(t *testing.T) {
	setup()

	/*
		testcases := []struct {
			got  interface{}
			want interface{}
		}{
			{
				AttributeQuery{Value: "68b329da9893e34099c7d8ad5cb9c940"},
				{},
			},
		}
	*/
	want := &AttributeQuery{Value: "68b329da9893e34099c7d8ad5cb9c940"}

	mux.HandleFunc("/attributes/restSearch/json/",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")
			d := json.NewDecoder(r.Body)

			var got AttributeQuery
			if err := d.Decode(&got); err != nil {
				t.Errorf("Cannot decode json SearchQuery request: %s", err)
			}

			if !reflect.DeepEqual(want, got) {
				t.Errorf("SearchAttribute returned %+v, want %+v", got, want)
			}

			fmt.Fprint(w, `{"id":1}`)
		})

	matches, err := client.SearchAttribute(want)
	if err != nil {
		t.Errorf("SearchAttribute returned error: %v", err)
	}

}

func Test_UploadSample(t *testing.T) {
	setup()

	s := &SampleUpload{
		Files: []SampleFile{
			{Filename: "foo", Data: "bar"},
		},
		Distribution: 2,
		EventID:      3,
		Comment:      "foobar",
		ToID:         false,
		Category:     "toto",
		Info:         "baz",
	}

	mux.HandleFunc("/events/upload_sample/",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")
			d := json.NewDecoder(r.Body)

			var got Request
			if err := d.Decode(&got); err != nil {
				t.Errorf("Cannot decode json SampleInput request: %s", err)
			}

			orig := Request{Request: *s}
			buf, _ := json.Marshal(orig)
			var want Request
			json.Unmarshal(buf, &want)
			if !reflect.DeepEqual(want, got) {
				t.Errorf("UploadSample returned %+v, want %+v", got, want)
			}

			fmt.Fprint(w, `{"id":1}`)
		})

	_, err := client.UploadSample(s)
	if err != nil {
		t.Errorf("UploadSample returned error: %v", err)
	}

	/*
		want := &User{ID: Int(1)}
		if !reflect.DeepEqual(user, want) {
			t.Errorf("Users.Get returned %+v, want %+v", user, want)
		}
	*/
}
