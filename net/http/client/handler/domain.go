package handler

import (
	"fmt"
	"github.com/rafaeljusto/shelter/log"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

func init() {
	HandleFunc(regexp.MustCompile(`^/domain/([[:alnum:]]|\-|\.)+$`), HandleDomain)
}

func HandleDomain(w http.ResponseWriter, r *http.Request) {
	restAddress, err := retrieveRESTAddress()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error while retrieving the REST address. Details:", err)
		return
	}

	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error while reading request body in web client. Details:", err)
		return
	}

	request, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s%s", restAddress, r.RequestURI),
		strings.NewReader(string(content)),
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error creating a request in web client. Details:", err)
		return
	}

	request.Header.Set("Accept-Language", r.Header.Get("Accept-Language"))

	response, err := signAndSend(request, content)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error signing and sending a request in web client. Details:", err)
		return
	}

	if response.StatusCode != http.StatusCreated &&
		response.StatusCode != http.StatusNoContent &&
		response.StatusCode != http.StatusBadRequest {

		w.WriteHeader(http.StatusInternalServerError)
		log.Println(fmt.Sprintf("Unexepected status code %d from /domain result "+
			"in web client", response.StatusCode))
		return
	}

	if response.StatusCode != http.StatusBadRequest {
		w.WriteHeader(response.StatusCode)

	} else {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(response.StatusCode)

		if _, err := io.Copy(w, response.Body); err != nil {
			// Here we already set the response code, so the client will receive a OK result
			// without body
			log.Println("Error copying REST response to web client response. Details:", err)
			return
		}
	}
}
