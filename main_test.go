// main_test.go

package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var a App

const tableCreationQuery = `
CREATE TABLE IF NOT EXISTS photos
(
	id INT AUTO_INCREMENT PRIMARY KEY,
	title VARCHAR(100),
	description TEXT,
	filename VARCHAR(100),
	ori_link VARCHAR(255)
)`

func TestMain(m *testing.M) {
	a = App{}
	a.Initialize("austin", "123456", "go_mux_api")

	ensureTableExists()

	code := m.Run()

	clearTable()

	os.Exit(code)
}

func ensureTableExists() {
	_, err := a.DB.Exec(tableCreationQuery)
	if err != nil {
		log.Fatal(err)
	}
}

func clearTable() {
	a.DB.Exec("DELETE FROM photos")
	a.DB.Exec("ALTER TABLE photos AUTO_INCREMENT = 1")
}

func TestEmptyTable(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/photos", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	body := response.Body.String()
	if body != "[]" {
		t.Errorf("Expected an empty array. Got %s", body)
	}
}

func TestGetNonExistentPhoto(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/photo/100", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response.Code)

	var m map[string]string
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["error"] != "Photo not found" {
		t.Errorf("Expected the 'error' key of the response to be set to 'Photo not found'. Got '%s'", m["error"])
	}
}

func TestCreatePhoto(t *testing.T) {
	clearTable()

	payload := []byte(`{"title":"test photo title","description":"test photo description", "filename":"test_image.jpg", "ori_link":"testorilink.com"`)

	req, _ := http.NewRequest("POST", "/photo", bytes.NewBuffer(payload))
	response := executeRequest(req)

	checkResponseCode(t, http.StatusCreated, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["title"] != "test photo title" {
		t.Errorf("Expected user name to be 'test photo title'. Got '%v'", m["title"])
	}

	if m["description"] != "test photo description" {
		t.Errorf("Expected user name to be 'test photo description'. Got '%v'", m["title"])
	}

	if m["filename"] != "test_image.jpg" {
		t.Errorf("Expected user name to be 'test_image.jpg'. Got '%v'", m["title"])
	}

	if m["ori_link"] != "testorilink.com" {
		t.Errorf("Expected user name to be 'testorilink.com'. Got '%v'", m["title"])
	}

	// the id is compared to 1.0 because JSON unmarshling converts numbers to floats,
	// when the target is a map[string]interface{}
	if m["id"] != 1.0 {
		t.Errorf("Expected photo ID to be '1'. Got '%v'", m["id"])
	}
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)
	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
