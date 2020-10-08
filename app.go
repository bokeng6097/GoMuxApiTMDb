package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
}

// Use *App to avoid copy of struct
func (a *App) Initialize(user, password, dbname string) {
	connectionString := fmt.Sprintf("%s:%s@/%s", user, password, dbname)

	var err error
	a.DB, err = sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	a.Router = mux.NewRouter()
	a.initializeRoutes()
}

func (a *App) Run(addr string) {
	log.Fatal(http.ListenAndServe(addr, a.Router))
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/photos", a.getPhotos).Methods("GET")
	a.Router.HandleFunc("/photo", a.createPhoto).Methods("POST")
	a.Router.HandleFunc("/photo/{id:[0-9]+}", a.getPhoto).Methods("GET")
	a.Router.HandleFunc("/photo/{id:[0-9]+}", a.updatePhoto).Methods("PUT")
	a.Router.HandleFunc("/photo/{id:[0-9]+}", a.deletePhoto).Methods("DELETE")
	a.Router.PathPrefix("/image/").Handler(http.StripPrefix("/image/", http.FileServer(http.Dir("image/"))))
}

func (a *App) getPhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid photo ID")
		return
	}

	p := photo{ID: id}
	if err := p.getPhoto(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "Photo not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	p.File = "localhost:8080/image/" + p.Filename
	respondWithJSON(w, http.StatusOK, p)
}

func (a *App) getPhotos(w http.ResponseWriter, r *http.Request) {
	photos, err := getPhotos(a.DB)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, photos)
}

func (a *App) createPhoto(w http.ResponseWriter, r *http.Request) {
	var p photo

	file, fileheader, err := r.FormFile("file")

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	defer file.Close()

	extension := filepath.Ext(fileheader.Filename)

	tempFile, err := ioutil.TempFile("image", "upload-*"+extension)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}

	// decoder := json.NewDecoder(r.Body)
	// if err := decoder.Decode(&p); err != nil {
	// 	fmt.Println(err)
	// 	respondWithError(w, http.StatusBadRequest, "Invalid request payload")
	// 	return
	// }

	// defer r.Body.Close()

	p.Title = r.FormValue("title")
	p.Description = r.FormValue("description")
	p.Filename = strings.Replace(tempFile.Name(), "image\\", "", -1)
	p.File = "localhost:8080/image/" + p.Filename
	p.OriLink = r.FormValue("ori_link")

	if err := p.createPhoto(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, p)
}

func (a *App) updatePhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid photo ID")
		return
	}

	var p photo

	file, fileheader, err := r.FormFile("file")

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	defer file.Close()

	extension := filepath.Ext(fileheader.Filename)

	tempFile, err := ioutil.TempFile("image", "upload-*"+extension)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	p.ID = id

	if err := p.getPhoto(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "Photo not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if err := os.Remove("image/" + p.Filename); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}

	p.Title = r.FormValue("title")
	p.Description = r.FormValue("description")
	p.Filename = strings.Replace(tempFile.Name(), "image\\", "", -1)
	p.File = "localhost:8080/image/" + p.Filename
	p.OriLink = r.FormValue("ori_link")

	if err := p.updatePhoto(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, p)
}

func (a *App) deletePhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid photo ID")
		return
	}

	var p photo
	p.ID = id

	if err := p.getPhoto(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "Photo not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if err := os.Remove("image/" + p.Filename); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}

	if err := p.deletePhoto(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
