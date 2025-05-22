package main

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMainHandler(t *testing.T) {
	setDB()
	type want struct {
		code        int
		respBody    string
		contentType string
	}
	tests := []struct {
		name    string
		path    string
		reqBody string
		method  string
		want    want
	}{
		{
			name:    "all good post",
			path:    "/",
			reqBody: `ya.ru`,
			method:  http.MethodPost,
			want: want{
				code:        http.StatusCreated,
				respBody:    `localhost:8080/21`,
				contentType: "text/plain",
			},
		},
		{
			name:    "all good get",
			path:    "/1",
			method:  http.MethodGet,
			reqBody: "",
			want: want{
				code:        http.StatusTemporaryRedirect,
				respBody:    "'kek'",
				contentType: "text/plain",
			},
		},
		{
			name:    "wrong id",
			path:    "/users",
			method:  http.MethodGet,
			reqBody: "ya.ru",
			want: want{
				code:        http.StatusBadRequest,
				respBody:    "",
				contentType: "",
			},
		},
		{
			name:    "wrong id",
			path:    "/26",
			method:  http.MethodGet,
			reqBody: "",
			want: want{
				code:        http.StatusBadRequest,
				respBody:    "",
				contentType: "",
			},
		},
		{
			name:    "all good JSON",
			path:    "/api/shorten",
			reqBody: `{"url":"ya.ru"}`,
			method:  http.MethodPost,
			want: want{
				code:        http.StatusCreated,
				respBody:    `{"result":"localhost:8080/2, "id":2}`,
				contentType: "application/json",
			},
		},
		{
			name:    "all good JSON batch",
			path:    "/api/shorten/batch",
			reqBody: `[{"url":"way1"}, {"url":"way2"}]`,
			method:  http.MethodPost,
			want: want{
				code:        http.StatusCreated,
				respBody:    `[{"result":"localhost:8080/17", "id":22}, {"result":"localhost:8080/18", "id":23}]`,
				contentType: "application/json",
			},
		},
		{
			name:    "all good DB ping",
			path:    "/ping",
			reqBody: ``,
			method:  http.MethodGet,
			want: want{
				code:        http.StatusOK,
				respBody:    ``,
				contentType: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/{id}", getLinkById)
			r.Post(`/`, reduceLink)
			r.Post(`/api/shorten`, reduceLinkJSON)
			r.Get(`/ping`, pingHandler)
			r.Post("/api/shorten/batch", reduceLinksBatchJSON)
			ts := httptest.NewServer(r)
			request, err := http.NewRequest(tt.method, ts.URL+tt.path, strings.NewReader(tt.reqBody))
			fmt.Println(ts.URL + tt.path)
			assert.NoError(t, err)
			res, err := ts.Client().Do(request)
			// проверяем код ответа
			assert.NoError(t, err)
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			resBody, err := io.ReadAll(res.Body)

			assert.NoError(t, err)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.respBody, string(resBody))
			defer res.Body.Close()
		})
	}
}
