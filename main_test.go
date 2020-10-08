// main_test.go

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
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

	file, _ := os.Open("conf.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := Config{}
	err := decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	a.Initialize(config.User, config.Password, config.Dbname)

	ensureTableExists()

	code := m.Run()

	clearTable()

	os.Exit(code)
}

func ensureTableExists() {
	if _, err := a.DB.Exec(tableCreationQuery); err != nil {
		log.Fatal(err)
	}
}

func clearTable() {
	a.DB.Exec("DELETE FROM photos")

	dir, err := ioutil.ReadDir("./image")
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range dir {
		os.RemoveAll(path.Join([]string{"image", d.Name()}...))
	}
	a.DB.Exec("ALTER TABLE photos AUTO_INCREMENT = 1")
}

func TestEmptyTable(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/photos", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); body != "[]" {
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
	var requestBody bytes.Buffer

	multiPartWriter := multipart.NewWriter(&requestBody)

	absPath, _ := filepath.Abs("./testdata/test_image.jpg")

	file, err := os.Open(absPath)
	if err != nil {
		t.Error(err)
	}
	defer file.Close()

	fileWriter, err := multiPartWriter.CreateFormFile("file", "test_image.jpg")
	if err != nil {
		t.Error(err)
	}

	if _, err := io.Copy(fileWriter, file); err != nil {
		t.Error(err)
	}

	if err := multiPartWriter.WriteField("title", "test title"); err != nil {
		t.Error(err)
	}

	if err := multiPartWriter.WriteField("description", "test description"); err != nil {
		t.Error(err)
	}

	if err := multiPartWriter.WriteField("ori_link", "test ori_link"); err != nil {
		t.Error(err)
	}

	multiPartWriter.Close()

	req, _ := http.NewRequest("POST", "/photo", &requestBody)
	req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())
	response := executeRequest(req)

	checkResponseCode(t, http.StatusCreated, response.Code)

	var p photo
	json.Unmarshal(response.Body.Bytes(), &p)

	if p.Title != "test title" {
		t.Errorf("Expected user name to be 'test title'. Got '%s'", p.Title)
	}

	if p.Description != "test description" {
		t.Errorf("Expected user name to be 'test description'. Got '%s'", p.Description)
	}

	if fileExists(t, p.Filename) == false {
		t.Errorf("Expected photo to be uploaded. Failed to detect photo with file name '%s'", p.Filename)
	}

	if p.OriLink != "test ori_link" {
		t.Errorf("Expected user name to be 'test ori_link'. Got '%s'", p.OriLink)
	}

	// the id is compared to 1.0 because JSON unmarshling converts numbers to floats,
	// when the target is a map[string]interface{}
	if p.ID != 1.0 {
		t.Errorf("Expected photo ID to be '1'. Got '%d'", p.ID)
	}

}

func TestUpdatePhoto(t *testing.T) {
	clearTable()
	addPhoto(t)

	req, _ := http.NewRequest("GET", "/photo/1", nil)
	response := executeRequest(req)
	var originalPhoto photo
	json.Unmarshal(response.Body.Bytes(), &originalPhoto)

	var requestBody bytes.Buffer

	multiPartWriter := multipart.NewWriter(&requestBody)

	absPath, _ := filepath.Abs("./testdata/test_image.jpg")

	file, err := os.Open(absPath)
	if err != nil {
		t.Error(err)
	}
	defer file.Close()

	fileWriter, err := multiPartWriter.CreateFormFile("file", "test_image.jpg")
	if err != nil {
		t.Error(err)
	}

	if _, err := io.Copy(fileWriter, file); err != nil {
		t.Error(err)
	}

	if err := multiPartWriter.WriteField("title", "test updated title"); err != nil {
		t.Error(err)
	}

	if err := multiPartWriter.WriteField("description", "test updated description"); err != nil {
		t.Error(err)
	}

	if err := multiPartWriter.WriteField("ori_link", "test updated ori_link"); err != nil {
		t.Error(err)
	}

	multiPartWriter.Close()

	req, _ = http.NewRequest("PUT", "/photo/1", &requestBody)
	req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())
	response = executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	var updatedPhoto photo
	json.Unmarshal(response.Body.Bytes(), &updatedPhoto)

	if updatedPhoto.ID != originalPhoto.ID {
		t.Errorf("Expected the id to remain the same '%d'. Got '%d'", originalPhoto.ID, updatedPhoto.ID)
	}

	if updatedPhoto.Title == originalPhoto.Title {
		t.Errorf("Expected the title to change from '%s' to 'test updated title'. Got '%s'", originalPhoto.Title, updatedPhoto.Title)
	}

	if updatedPhoto.Description == originalPhoto.Description {
		t.Errorf("Expected the description to change from '%s' to 'test updated description'. Got '%s'", originalPhoto.Description, updatedPhoto.Description)
	}

	if updatedPhoto.Filename == originalPhoto.Filename {
		t.Errorf("Expected the file name to change from '%s' to a new file name. Got '%s'", originalPhoto.Filename, updatedPhoto.Filename)
	}

	if fileExists(t, updatedPhoto.Filename) == false {
		t.Errorf("Expected photo to be uploaded. Failed to detect photo with file name '%s'", updatedPhoto.Filename)
	}

	if updatedPhoto.OriLink == originalPhoto.OriLink {
		t.Errorf("Expected the ori_link to change from '%s' to 'test updated ori_link'. Got '%s'", originalPhoto.OriLink, updatedPhoto.OriLink)
	}

}

func TestDeletePhoto(t *testing.T) {
	clearTable()
	addPhoto(t)

	req, _ := http.NewRequest("GET", "/photo/1", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest("DELETE", "/photo/1", nil)
	response = executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest("GET", "/photo/1", nil)
	response = executeRequest(req)
	checkResponseCode(t, http.StatusNotFound, response.Code)
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

func fileExists(t *testing.T, filename string) bool {
	absPath, _ := filepath.Abs("./image/" + filename)

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		t.Error(err)
		return false
	}
	return !info.IsDir()
}

func addPhoto(t *testing.T) {

	absPath, _ := filepath.Abs("./testdata/test_image.jpg")

	file, err := os.Open(absPath)
	if err != nil {
		t.Errorf(err.Error())
	}
	defer file.Close()

	tempFile, err := ioutil.TempFile("image", "upload-*.jpg")
	if err != nil {
		t.Errorf(err.Error())
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		t.Errorf(err.Error())
	}

	statement := fmt.Sprintf("INSERT INTO photos(title, description, filename, ori_link) VALUES ('Title 1', 'Description 1', '%s', 'Ori link 1')", strings.Replace(tempFile.Name(), "image\\", "", -1))
	if _, err := a.DB.Exec(statement); err != nil {
		t.Errorf(err.Error())
	}
}
