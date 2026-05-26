package health

import (
	"encoding/json"
	"net/http"
)

// Responds with a 200 OK and a JSON body of {"status":"OK"}.
func HealthHandler(wtr http.ResponseWriter, req *http.Request) {
	wtr.Header().Set("Content-Type", "application/json")
	wtr.WriteHeader(http.StatusOK)
	json.NewEncoder(wtr).Encode(map[string]string{"status": "OK"})
}
