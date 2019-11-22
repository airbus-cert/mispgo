package misp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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

type attributeRequest struct {
	Request AttributeQuery
}

func Test_AddSightingNotFound(t *testing.T) {
	setup()

	mux.HandleFunc("/sightings/add/",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")
			w.WriteHeader(403)
			fmt.Fprint(w,
				`{
    "name": "Could not add Sighting",
    "message": "Could not add Sighting",
    "url": "\/sightings\/add",
    "errors": "No valid attributes found that match the criteria."
}`)
		})
	_, err := client.AddSighting(&Sighting{Value: "NOT FOUND"})
	if err == nil {
		t.Errorf("AddSighting() did not returned an error, I was expecting status=403")
	}

}

func Test_AddSighting(t *testing.T) {
	setup()
	mux.HandleFunc("/sightings/add/",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")

			fmt.Fprint(w, `{"name": "2 sightings successfuly added.", "message": "2 sightings successfuly added.", "url": "\/sightings\/add"}`)
		})

	_, err := client.AddSighting(&Sighting{Value: "foobar.com"})
	if err != nil {
		t.Errorf("AddSighting() failed: %v", err)
	}

}

func Test_SearchAttribute_NoResult(t *testing.T) {
	setup()

	attrReq := &AttributeQuery{Value: "68b329da9893e34099c7d8ad5cb9c940"}
	mux.HandleFunc("/attributes/restSearch/json/",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"response":[]}`)
		})

	_, err := client.SearchAttribute(attrReq)
	if err != nil {
		t.Errorf("SearchAttribute returned error: %v", err)
	}
}

func Test_SearchAttribute(t *testing.T) {
	setup()

	attrReq := &AttributeQuery{Value: "68b329da9893e34099c7d8ad5cb9c940"}
	want := attributeRequest{Request: *attrReq}

	mux.HandleFunc("/attributes/restSearch/json/",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")
			d := json.NewDecoder(r.Body)

			var got attributeRequest
			if err := d.Decode(&got); err != nil {
				t.Errorf("Cannot decode json SearchQuery request: %s", err)
			}

			if !reflect.DeepEqual(want, got) {
				t.Errorf("SearchAttribute returned %+v, want %+v", got, want)
			}

			fmt.Fprint(w, `{"response":{"Attribute":[{"id":"610744","event_id":"6871","category":"Payload delivery","type":"filename|md5","to_ids":true,"uuid":"58b98766-73cc-437f-a814-4a9a0a3ac101","timestamp":"1488553830","distribution":"5","comment":"my comment 1","sharing_group_id":"0","deleted":false,"disable_correlation":true,"object_id":"0","object_relation":null,"value":"1.bat|68b329da9893e34099c7d8ad5cb9c940"},{"id":"610783","event_id":"6871","category":"Artifacts dropped","type":"md5","to_ids":true,"uuid":"58b98dc1-b698-4172-b274-4ae30a3ac101","timestamp":"1488557887","distribution":"5","comment":"1.bat","sharing_group_id":"0","deleted":false,"disable_correlation":false,"object_id":"0","object_relation":null,"value":"68b329da9893e34099c7d8ad5cb9c940"}]}}`)
		})

	matches, err := client.SearchAttribute(attrReq)
	if err != nil {
		t.Errorf("SearchAttribute returned error: %v", err)
	}

	attributesWanted := []Attribute{
		{
			Comment:            "my comment 1",
			ID:                 "610744",
			EventID:            "6871",
			Category:           "Payload delivery",
			Type:               "filename|md5",
			ToIDS:              true,
			UUID:               "58b98766-73cc-437f-a814-4a9a0a3ac101",
			Timestamp:          "1488553830",
			Distribution:       "5",
			SharingGroupID:     "0",
			Deleted:            false,
			DisableCorrelation: true,
			ObjectID:           "0",
			ObjectRelation:     "",
			Value:              "1.bat|68b329da9893e34099c7d8ad5cb9c940",
		},
		{
			Comment:            "1.bat",
			ID:                 "610783",
			EventID:            "6871",
			Category:           "Artifacts dropped",
			Type:               "md5",
			ToIDS:              true,
			UUID:               "58b98dc1-b698-4172-b274-4ae30a3ac101",
			Timestamp:          "1488557887",
			Distribution:       "5",
			SharingGroupID:     "0",
			Deleted:            false,
			DisableCorrelation: false,
			ObjectID:           "0",
			ObjectRelation:     "",
			Value:              "68b329da9893e34099c7d8ad5cb9c940",
		},
	}

	if !reflect.DeepEqual(matches, attributesWanted) {
		t.Errorf("Search results were different than expected: got %v, wanted %v", matches, attributesWanted)
	}

}

func Test_UploadSample_Failed(t *testing.T) {
	setup()

	s := &SampleUpload{
		Files: []SampleFile{
			{Filename: "foo", Data: "bar"},
		},
		Distribution: "5",
		EventID:      "3",
		Comment:      "foobar",
		ToIDS:        false,
		Category:     "toto",
		Info:         "baz",
	}

	mux.HandleFunc("/events/upload_sample/",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")
			fmt.Fprintf(w, `{"url": "/events/upload_sample", "message": "Distribution level 5 is not supported when uploading a sample without passing an event ID. Distribution level 5 is meant to take on the distribution level of an existing event.", "errors": ["Distribution level 5 is not supported when uploading a sample without passing an event ID. Distribution level 5 is meant to take on the distribution level of an existing event."], "name": "Distribution level 5 is not supported when uploading a sample without passing an event ID. Distribution level 5 is meant to take on the distribution level of an existing event."}`)

		})

	_, err := client.UploadSample(s)
	if err == nil {
		t.Errorf("UploadSample returned error: %v", err)
	}
}

func Test_UploadSample(t *testing.T) {
	setup()

	s := &SampleUpload{
		Files: []SampleFile{
			{Filename: "foo", Data: "bar"},
		},
		Distribution: "2",
		EventID:      "3",
		Comment:      "foobar",
		ToIDS:        false,
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

			fmt.Fprint(w, `{"url": "/events/view/11169", "message": "Success, saved all attributes.", "name": "Success", "id": "11169"}`)
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

func Test_AddEventTag(t *testing.T) {
	setup()

	handler := func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		d := json.NewDecoder(r.Body)

		var got EventTag
		if err := d.Decode(&got); err != nil {
			t.Errorf("Cannot decode json EventTag request: %s", err)
		}

		if got.Event.ID != "666" {
			t.Errorf("Decoding EventTag.Event.ID failed, expected 666 got %#v", got.Event.ID)
		}

		if got.Event.Tag != "TLP:AMBER" {
			t.Errorf("Decoding EventTag.Event.Tag failed, expected TLP:AMBER got %#v", got.Event.Tag)
		}
	}
	mux.HandleFunc("/events/addTag", handler)
	mux.HandleFunc("/events/removeTag", handler)

	_, err := client.AddEventTag("666", "TLP:AMBER")
	if err != nil {
		t.Errorf("Error while adding EventTag: %#v", err)
	}

	_, err = client.RemoveEventTag("666", "TLP:AMBER")
	if err != nil {
		t.Errorf("Error while removing EventTag: %#v", err)
	}
}

//func Test_RemoveEventTag

func Test_DownloadSample(t *testing.T) {
	setup()

	mux.HandleFunc("/attributes/downloadAttachment/download/1234",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")

			w.Write([]byte{0xAB, 0xCD, 0xEF, 0x13, 0x37})
		})

	err := client.DownloadSample(1234, "test_DownloadSample.bin")
	if err != nil {
		t.Errorf("DownloadSample returned an error: %s", err)
	}
	defer os.Remove("test_DownloadSample.bin")

	result, err := ioutil.ReadFile("test_DownloadSample.bin")
	if err != nil {
		t.Errorf("ReadFile returned an error: %s", err)
	}

	expected := []byte{0xAB, 0xCD, 0xEF, 0x13, 0x37}
	if !bytes.Equal(expected, result) {
		t.Errorf("Wrong download:\n\texpected %#v\n\tgot %#v", expected, result)
	}
}

func TestAddAttribute(t *testing.T) {
	setup()

	attr := Attribute{
		Value:    "1.2.3.4",
		Type:     "ip-dst",
		Category: "Network activity",
	}

	mux.HandleFunc("/attributes/add/1234",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")
			d := json.NewDecoder(r.Body)

			var got Attribute
			if err := d.Decode(&got); err != nil {
				t.Errorf("Cannot decode json AddAttribute request: %s", err)
			}

			if !reflect.DeepEqual(got, attr) {
				t.Errorf("AddAttribute returned %+v, want %+v", got, attr)
			}

			fmt.Fprint(w, `
			{
				"Attribute": {
					"id": "3993961",
					"event_id": "1234",
					"object_id": "0",
					"object_relation": null,
					"category": "Network activity",
					"type": "ip-dst",
					"value1": "1.2.3.4",
					"value2": "",
					"to_ids": true,
					"uuid": "5dd790ad-b0ec-4b8a-bc97-2ed00a3a5cd9",
					"timestamp": "1574408365",
					"distribution": "5",
					"sharing_group_id": "0",
					"comment": "",
					"deleted": false,
					"disable_correlation": false,
					"value": "1.2.3.4"
				}
			}
			`)
		})

	newAttr, err := client.AddAttribute("1234", attr)
	if err != nil {
		t.Errorf("AddAttribute returned an error: %s", err)
	}

	if newAttr.EventID != "1234" {
		t.Errorf("Returned EventID attribute does not match: got %v, expecting %v", newAttr.EventID, attr.EventID)
	}

	if newAttr.Value != attr.Value {
		t.Errorf("Returned Value attribute does not match: got %v, expecting %v", newAttr.Value, attr.Value)
	}

	if newAttr.Category != attr.Category {
		t.Errorf("Returned Category attribute does not match: got %v, expecting %v", newAttr.Category, attr.Category)
	}

	if newAttr.Type != attr.Type {
		t.Errorf("Returned Type attribute does not match: got %v, expecting %v", newAttr.Type, attr.Type)
	}
}
