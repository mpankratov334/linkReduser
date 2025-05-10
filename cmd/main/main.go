package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var links = []string{``}

type jsonReq struct {
	Url string `json:"url"`
}
type jsonLink struct {
	Id          int    `json:"id"`
	OriginalUrl string `json:"original_url"`
	ShortUrl    string `json:"short_url"`
}
type jsonResp struct {
	Result string `json:"result"`
	Id     int    `json:"id"`
}
type jsonUserLinks struct {
	Short    string `json:"short_url"`
	Original string `json:"original_url"`
}

func reduceLinkJSON(w http.ResponseWriter, req *http.Request) {

	var (
		jsr     jsonReq
		buf     bytes.Buffer
		newLink string
		link    string
		id      int
		parts   []string
	)

	if _, err := buf.ReadFrom(req.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &jsr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	link = jsr.Url
	userID, ok := req.Context().Value(USERID_KEY).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	newLink, err := writeUrlInDB([]byte(link), userID)
	if errors.Is(err, errors.New("conflict")) {
		w.WriteHeader(http.StatusConflict)
	} else if err != nil {
		w.WriteHeader(http.StatusCreated)
	}
	parts = strings.Split(newLink, "/")
	id, _ = strconv.Atoi(parts[len(parts)-1])
	resp, err := json.Marshal(jsonResp{Result: newLink, Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func reduceLink(w http.ResponseWriter, req *http.Request) {
	last_id := len(links)
	link, _ := io.ReadAll(req.Body)
	links = append(links, string(link))

	var (
		newLink string
		err     error
	)
	if storeFilePath != "" {
		newLink = storeUrlInFile(link, last_id)
		w.Header().Set("content-type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(newLink))
	} else {
		userID, ok := req.Context().Value(USERID_KEY).(int)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		newLink, err = writeUrlInDB(link, userID)

		if err != nil {
			w.Header().Set("content-type", "text/plain")
			w.WriteHeader(http.StatusConflict)
		} else {
			w.Header().Set("content-type", "text/plain")
			w.WriteHeader(http.StatusCreated)
		}
	}
	w.Write([]byte(newLink))

}

func getLinkById(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("content-type", "text/plain")
	id, err := strconv.ParseInt(chi.URLParam(req, "id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var link string
	if storeFilePath != "" {
		if int(id) > len(links)-1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		link = links[id]
	} else {
		link, err = getLinkFromDB(id)
		if errors.Is(err, errors.New("deleted")) {
			w.WriteHeader(http.StatusGone)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	w.WriteHeader(http.StatusTemporaryRedirect)
	w.Write([]byte(link))
}

func reduceLinksBatchJSON(w http.ResponseWriter, req *http.Request) {
	var jsreqs []jsonReq
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(req.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		fmt.Println("req body")
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &jsreqs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		fmt.Println("error in unmarshalling request")
		return
	}
	var linkId int
	db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM Links").Scan(&linkId)
	var jsrps []jsonResp
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	for i := 0; i < len(jsreqs); i++ {
		linkId++
		newLink := saveAdr + `/` + strconv.Itoa(linkId)
		jsrps = append(jsrps, jsonResp{Result: newLink, Id: linkId})
	}
	resp, err := json.Marshal(jsrps)
	if err != nil {
		fmt.Println("error in marshaling response")
		fmt.Println(jsrps)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = insertLinksDB(req.Context(), db, jsreqs)
	if err != nil {
		if errors.Is(err, errors.New("Unauthorized")) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	w.Write(resp)
}

func getLinkByUser(w http.ResponseWriter, req *http.Request) {
	userId, ok := req.Context().Value(USERID_KEY).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	rows, err := getJsonLinkByUserID(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	var jsonResponce []jsonUserLinks
	for rows.Next() {
		var link jsonUserLinks
		err := rows.Scan(&link.Short, &link.Original)
		if err != nil {
			http.Error(w, fmt.Sprintf("Row scan error: %v", err), http.StatusInternalServerError)
			return
		}
		jsonResponce = append(jsonResponce, link)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Rows error: %v", err), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(jsonResponce)
	if err != nil {
		http.Error(w, fmt.Sprintf("JSON error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func deleteLinksJSON(w http.ResponseWriter, req *http.Request) {
	var linksId []string
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(req.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &linksId); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	if err := deleteLinks(req.Context(), linksId); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func main() {
	con := setConfig()
	if con.storeFile != "" {
		setFileStorage()
	} else {
		setDB()
	}
	setLogger()
	r := chi.NewRouter()
	r.Get("/{id}", WithLogging(getLinkById))
	r.Post(`/`, WithAuthorization(gzipHandle(WithLogging(reduceLink))))
	r.Post(`/api/shorten`, WithAuthorization(gzipHandle(WithLogging(reduceLinkJSON))))
	r.Get(`/ping`, WithLogging(pingHandler))
	r.Post(`/api/shorten/batch`, WithAuthorization(WithLogging(reduceLinksBatchJSON)))
	r.Get(`/api/user/urls`, WithAuthorization(WithLogging(getLinkByUser)))
	r.Delete(`/api/users/urls`, WithAuthorization(gzipHandle(WithLogging(deleteLinksJSON))))
	err := http.ListenAndServe(adr, r)
	if err != nil {
		panic(err)
	}
}
