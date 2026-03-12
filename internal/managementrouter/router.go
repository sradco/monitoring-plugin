package managementrouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"

	"github.com/openshift/monitoring-plugin/pkg/management"
)

type httpRouter struct {
	managementClient management.Client
}

func New(managementClient management.Client) *mux.Router {
	httpRouter := &httpRouter{
		managementClient: managementClient,
	}

	r := mux.NewRouter()

	r.HandleFunc("/api/v1/alerting/rules", httpRouter.CreateAlertRule).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/alerting/rules", httpRouter.BulkDeleteUserDefinedAlertRules).Methods(http.MethodDelete)
	r.HandleFunc("/api/v1/alerting/rules", httpRouter.BulkUpdateAlertRules).Methods(http.MethodPatch)

	return r
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp, err := json.Marshal(map[string]string{"error": message})
	if err != nil {
		log.Printf("failed to marshal error response: %v", err)
		return
	}
	if _, err := w.Write(resp); err != nil {
		log.Printf("failed to write error response: %v", err)
	}
}

func handleError(w http.ResponseWriter, err error) {
	status, message := parseError(err)
	writeError(w, status, message)
}

func parseError(err error) (int, string) {
	var nf *management.NotFoundError
	if errors.As(err, &nf) {
		return http.StatusNotFound, err.Error()
	}
	var ve *management.ValidationError
	if errors.As(err, &ve) {
		return http.StatusBadRequest, err.Error()
	}
	var na *management.NotAllowedError
	if errors.As(err, &na) {
		return http.StatusMethodNotAllowed, err.Error()
	}
	var ce *management.ConflictError
	if errors.As(err, &ce) {
		return http.StatusConflict, err.Error()
	}
	log.Printf("An unexpected error occurred: %v", err)
	return http.StatusInternalServerError, "An unexpected error occurred"
}

func parseParam(raw string, name string) (string, error) {
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return "", fmt.Errorf("invalid %s encoding", name)
	}
	value := strings.TrimSpace(decoded)
	if value == "" {
		return "", fmt.Errorf("missing %s", name)
	}
	return value, nil
}
