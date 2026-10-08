package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/company/service-registry/internal/storage"
	"google.golang.org/protobuf/encoding/protojson"
)

func registerEventSSE(mux *http.ServeMux, eventRepo *storage.EventRepository) {
	mux.HandleFunc("/api/v1/events/watch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		filters := storage.EventFilters{
			EnvironmentID: firstQueryValue(r, "environmentId", "environment_id"),
			ServiceID:     firstQueryValue(r, "serviceId", "service_id"),
			DeploymentID:  firstQueryValue(r, "deploymentId", "deployment_id"),
			InstanceID:    firstQueryValue(r, "instanceId", "instance_id"),
		}
		cursor := time.Now().UTC().Add(-time.Nanosecond)
		if r.URL.Query().Get("replay") == "true" {
			cursor = time.Unix(0, 0).UTC()
		}
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			events, err := eventRepo.ListSince(r.Context(), filters, cursor, 50)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %q\n\n", "Unable to read events")
				flusher.Flush()
				return
			}
			for _, event := range events {
				if event.GetTimestamp() != nil {
					cursor = event.GetTimestamp().AsTime().UTC()
				}
				payload, err := protojson.Marshal(event)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "event: registry-event\ndata: %s\n\n", payload)
				flusher.Flush()
			}

			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
			}
		}
	})
}

func firstQueryValue(r *http.Request, names ...string) string {
	for _, name := range names {
		if value := r.URL.Query().Get(name); value != "" {
			return value
		}
	}
	return ""
}
