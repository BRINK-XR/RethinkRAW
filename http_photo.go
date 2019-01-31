package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/schema"
)

func photoHandler(w http.ResponseWriter, r *http.Request) HTTPResult {
	path := r.URL.Path
	r.ParseForm()

	_, meta := r.Form["meta"]
	_, save := r.Form["save"]
	_, export := r.Form["export"]
	_, preview := r.Form["preview"]
	_, settings := r.Form["settings"]

	switch {
	case meta:
		if r := cacheHeaders(path, r.Header, w.Header()); r.Done() {
			return r
		}

		if out, err := getMeta(path); err != nil {
			return HTTPResult{Error: err}
		} else {
			w.Header().Set("Cache-Control", "max-age=60")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write(out)
			return HTTPResult{}
		}

	case save:
		var xmp xmpSettings
		xmp.Filename = path
		dec := schema.NewDecoder()
		dec.IgnoreUnknownKeys(true)
		if err := dec.Decode(&xmp, r.Form); err != nil {
			return HTTPResult{Error: err}
		}
		if err := saveEdit(path, &xmp); err != nil {
			return HTTPResult{Error: err}
		} else {
			return HTTPResult{Status: http.StatusNoContent}
		}

	case export:
		var xmp xmpSettings
		xmp.Filename = path
		var exp exportSettings
		dec := schema.NewDecoder()
		dec.IgnoreUnknownKeys(true)
		if err := dec.Decode(&xmp, r.Form); err != nil {
			return HTTPResult{Error: err}
		}
		if err := dec.Decode(&exp, r.Form); err != nil {
			return HTTPResult{Error: err}
		}
		if out, err := exportEdit(path, &xmp, &exp); err != nil {
			return HTTPResult{Error: err}
		} else {
			exportHeaders(path, &exp, w.Header())
			w.Write(out)
			return HTTPResult{}
		}

	case preview:
		var xmp xmpSettings
		dec := schema.NewDecoder()
		dec.IgnoreUnknownKeys(true)
		if err := dec.Decode(&xmp, r.Form); err != nil {
			return HTTPResult{Error: err}
		}
		if out, err := previewEdit(path, &xmp); err != nil {
			return HTTPResult{Error: err}
		} else {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write(out)
			return HTTPResult{}
		}

	case settings:
		if xmp, err := loadEdit(path); err != nil {
			return HTTPResult{Error: err}
		} else {
			w.Header().Set("Content-Type", "application/json")
			enc := json.NewEncoder(w)
			if err := enc.Encode(xmp); err != nil {
				return HTTPResult{Error: err}
			}
		}
		return HTTPResult{}

	default:
		if _, err := os.Stat(path); err != nil {
			return HTTPResult{Error: err}
		}

		w.Header().Set("Cache-Control", "max-age=300")
		w.Header().Set("Content-Type", "text/html")
		return HTTPResult{
			Error: templates.ExecuteTemplate(w, "photo.html", struct {
				Name, Title, Path string
			}{
				filepath.Base(path),
				filepath.Clean(path),
				toURLPath(filepath.Clean(path)),
			}),
		}
	}
}
