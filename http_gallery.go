package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	nfd "github.com/ncruces/go-nativefiledialog"
)

var extensions = map[string]struct{}{
	".CRW": {}, // Canon
	".NEF": {}, // Nikon
	".RAF": {}, // Fujifilm
	".ORF": {}, // Olympus
	".MRW": {}, // Minolta
	".DCR": {}, // Kodak
	".MOS": {}, // Leaf
	".RAW": {}, // Panasonic
	".PEF": {}, // Pentax
	".SRF": {}, // Sony
	".DNG": {}, // Adobe
	".X3F": {}, // Sigma
	".CR2": {}, // Canon
	".ERF": {}, // Epson
	".SR2": {}, // Sony
	".KDC": {}, // Kodak
	".MFW": {}, // Mamiya
	".MEF": {}, // Mamiya
	".ARW": {}, // Sony
	".NRW": {}, // Nikon
	".RW2": {}, // Panasonic
	".RWL": {}, // Leica
	".IIQ": {}, // Phase One
	".3FR": {}, // Hasselblad
	".FFF": {}, // Hasselblad
	".SRW": {}, // Samsung
	".GPR": {}, // GoPro
	".DXO": {}, // DxO
	".ARQ": {}, // Sony
	".CR3": {}, // Canon
}

func galleryHandler(w http.ResponseWriter, r *http.Request) HTTPResult {
	path := r.URL.Path
	r.ParseForm()

	_, browse := r.Form["browse"]
	if browse {
		bringToTop()
		if folder, err := nfd.PickFolder(path); err != nil {
			return HTTPResult{Error: err}
		} else if folder == "" {
			return HTTPResult{Status: http.StatusResetContent}
		} else if fi, err := os.Stat(folder); os.IsNotExist(err) {
			return HTTPResult{Status: http.StatusResetContent}
		} else if err != nil {
			return HTTPResult{Error: err}
		} else {
			var url url.URL
			if fi.IsDir() {
				url.Path = "/gallery/" + toURLPath(folder)
			} else {
				url.Path = "/photo/" + toURLPath(folder)
			}
			return HTTPResult{
				Status:   http.StatusSeeOther,
				Location: url.String(),
			}
		}
	}

	if r := cacheHeaders(path, r.Header, w.Header()); r.Done() {
		return r
	}

	if files, err := ioutil.ReadDir(path); err != nil {
		return HTTPResult{Error: err}
	} else {
		path = filepath.Join(path, ".")
		parent := filepath.Join(path, "..")
		if path == parent {
			parent = ""
		}

		data := struct {
			Title, Parent string
			Dirs, Photos  []struct{ Name, Path string }
		}{
			Title:  filepath.Clean(path),
			Parent: toURLPath(parent),
		}

		for _, i := range files {
			if isHidden(i) {
				continue
			}

			name := i.Name()
			item := struct{ Name, Path string }{name, toURLPath(filepath.Join(path, name))}

			if i.IsDir() {
				data.Dirs = append(data.Dirs, item)
			} else if _, ok := extensions[strings.ToUpper(filepath.Ext(name))]; ok {
				data.Photos = append(data.Photos, item)
			}
		}

		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("Content-Type", "text/html")
		return HTTPResult{
			Error: templates.ExecuteTemplate(w, "gallery.html", data),
		}
	}
}
