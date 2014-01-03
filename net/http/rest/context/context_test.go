package context

import (
	"errors"
	"math"
	"net/http"
	"shelter/net/http/rest/language"
	"strings"
	"testing"
)

type ReaderWithError struct{}

func (r *ReaderWithError) Read(p []byte) (n int, err error) {
	n = 0
	err = errors.New("Just throwing an error for tests")
	return
}

func TestNewShelterRESTContext(t *testing.T) {
	r, err := http.NewRequest("", "", strings.NewReader("Test"))
	if err != nil {
		t.Fatal(err)
	}

	context, err := NewShelterRESTContext(r, nil)
	if err != nil {
		t.Fatal(err)
	}

	if string(context.RequestContent) != "Test" {
		t.Error("Not storing request body correctly")
	}

	r, err = http.NewRequest("", "", &ReaderWithError{})
	if err != nil {
		t.Fatal(err)
	}

	r.ContentLength = 100
	_, err = NewShelterRESTContext(r, nil)
	if err == nil {
		t.Error("Not detecting request content error")
	}
}

func TestJSONRequest(t *testing.T) {
	r, err := http.NewRequest("", "",
		strings.NewReader("{\"key\": \"value\"}"))
	if err != nil {
		t.Fatal(err)
	}

	context, err := NewShelterRESTContext(r, nil)
	if err != nil {
		t.Fatal(err)
	}

	object := struct {
		Key string `json:"key"`
	}{
		Key: "",
	}

	if err := context.JSONRequest(&object); err != nil {
		t.Fatal(err)
	}

	if object.Key != "value" {
		t.Error("Not decoding a JSON object properly")
	}
}

func TestResponse(t *testing.T) {
	r, err := http.NewRequest("", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	context, err := NewShelterRESTContext(r, nil)
	if err != nil {
		t.Fatal(err)
	}

	context.Response(http.StatusNotFound)
	if context.ResponseHTTPStatus != http.StatusNotFound {
		t.Error("Not setting the return status code properly")
	}
}

func TestMessageReponse(t *testing.T) {
	r, err := http.NewRequest("", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	context, err := NewShelterRESTContext(r, nil)
	if err != nil {
		t.Fatal(err)
	}

	context.Language = &language.LanguagePack{
		Messages: map[string]string{
			"key": "value",
		},
	}

	context.MessageResponse(http.StatusNotFound, "key")

	if context.ResponseHTTPStatus != http.StatusNotFound {
		t.Error("Not setting the return status code properly")
	}

	if string(context.ResponseContent) != "value" {
		t.Error("Not setting the return message properly")
	}
}

func TestJSONReponse(t *testing.T) {
	r, err := http.NewRequest("", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	context, err := NewShelterRESTContext(r, nil)
	if err != nil {
		t.Fatal(err)
	}

	object := struct {
		Key string `json:"key"`
	}{
		Key: "value",
	}
	if err := context.JSONResponse(http.StatusNotFound, object); err != nil {
		t.Fatal("Not creating a valid JSON")
	}

	if context.ResponseHTTPStatus != http.StatusNotFound {
		t.Error("Not setting the return status code properly")
	}

	if string(context.ResponseContent) != "{\"key\":\"value\"}" {
		t.Error("Not setting the return message properly")
	}

	if err := context.JSONResponse(http.StatusOK, math.NaN()); err == nil {
		t.Error("Not detecting strange JSON objects")
	}
}

func TestAddHeader(t *testing.T) {
	r, err := http.NewRequest("", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	context, err := NewShelterRESTContext(r, nil)
	if err != nil {
		t.Fatal(err)
	}

	context.AddHeader("Content-Type", "text/plain")
	if _, ok := context.HTTPHeader["Content-Type"]; ok {
		t.Error("Allowing fixed HTTP headers to be replaced")
	}

	context.AddHeader("ETag", "1")
	if value, ok := context.HTTPHeader["ETag"]; !ok || value != "1" {
		t.Error("Not storing HTTP custom header properly")
	}
}