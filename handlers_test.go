package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mozilla-services/go-bouncer/bouncer"
	"github.com/stretchr/testify/assert"
)

var bouncerHandler *BouncerHandler

func init() {
	testDB, err := bouncer.NewDB("root@tcp(127.0.0.1:3306)/bouncer_test")
	if err != nil {
		log.Fatal(err)
	}

	bouncerHandler = &BouncerHandler{db: testDB}
}

func TestBouncerHandlerParams(t *testing.T) {
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://test/?os=mac&lang=en-US", nil)
	assert.NoError(t, err)

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "http://www.mozilla.org/", w.HeaderMap.Get("Location"))
}

func TestBouncerHandlerPrintQuery(t *testing.T) {
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://test/?product=firefox-latest&os=osx&lang=en-US&print=yes", nil)
	assert.NoError(t, err)

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "http://download-installer.cdn.mozilla.net/pub/firefox/releases/39.0/mac/en-US/Firefox%2039.0.dmg", w.Body.String())
}

func TestBouncerHandlerValid(t *testing.T) {
	bouncerHandler.godspeedChan = make(chan []string, 100)
	defer func() { bouncerHandler.godspeedChan = nil }()

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://test/?product=firefox-latest&os=osx&lang=en-US", nil)
	assert.NoError(t, err)

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "http://download-installer.cdn.mozilla.net/pub/firefox/releases/39.0/mac/en-US/Firefox%2039.0.dmg", w.HeaderMap.Get("Location"))

	tags, ok := <-bouncerHandler.godspeedChan
	assert.True(t, ok)
	assert.Contains(t, tags, "product:firefox-latest")
	assert.Contains(t, tags, "os:osx")
	assert.Contains(t, tags, "language:en-US")

	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "http://test/?product=firefox-latest&os=win64&lang=en-US", nil)
	assert.NoError(t, err)

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "http://download-installer.cdn.mozilla.net/pub/firefox/releases/39.0/win32/en-US/Firefox%20Setup%2039.0.exe", w.HeaderMap.Get("Location"))

	tags, ok = <-bouncerHandler.godspeedChan
	assert.True(t, ok)
	assert.Contains(t, tags, "product:firefox-latest")
	assert.Contains(t, tags, "os:win64")
	assert.Contains(t, tags, "language:en-US")

	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "http://test/?product=Firefox-SSL&os=win64&lang=en-US", nil)
	assert.NoError(t, err)

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "https://download-installer.cdn.mozilla.net/pub/firefox/releases/39.0/win32/en-US/Firefox%20Setup%2039.0.exe", w.HeaderMap.Get("Location"))

	tags, ok = <-bouncerHandler.godspeedChan
	assert.True(t, ok)
	assert.Contains(t, tags, "product:firefox-ssl")
	assert.Contains(t, tags, "os:win64")
	assert.Contains(t, tags, "language:en-US")
}
