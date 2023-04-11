package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/schema"
)

func serverBatchHandler(w http.ResponseWriter, r *http.Request) httpResult {
	if err := r.ParseForm(); err != nil {
		return httpResult{Status: http.StatusBadRequest, Error: err}
	}

	_, path := r.Form["path"]
	_, export := r.Form["export"]

	switch {
	case path:
		paths := r.Form["filepath[]"]
		batchPath := toBatchPath(paths...)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, batchPath)
		return httpResult{}

	case export:
		currentPath := r.URL.Path
		batchPath := strings.Replace(currentPath, "/serverBatch/", "", -1)
		batch := fromBatchPath(batchPath)

		photos, err := findPhotos(batch)
		if err != nil {
			return httpResult{Error: err}
		}

		var xmp xmpSettings
		var exp exportSettings
		dec := schema.NewDecoder()
		dec.IgnoreUnknownKeys(true)
		if err := dec.Decode(&xmp, r.Form); err != nil {
			return httpResult{Error: err}
		}
		if err := dec.Decode(&exp, r.Form); err != nil {
			return httpResult{Error: err}
		}
		xmp.Orientation = 0

		exppath := r.Form["output"][0]
		if _, err := os.Stat(exppath); err != nil {
			return httpResult{Error: err}
		}

		results := batchProcess(r.Context(), photos, func(ctx context.Context, photo batchPhoto) error {
			err := batchProcessPhoto(ctx, photo, exppath, xmp, exp)
			if err == nil && exp.Both {
				err = batchProcessPhoto(ctx, photo, exppath, xmp, exportSettings{})
			}
			return err
		})

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusMultiStatus)
		batchResultWriter(w, results, len(photos))
		return httpResult{}
	}
	return httpResult{}
}
