package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var pHost = flag.String("host", "0.0.0.0", "http host")
var pPort = flag.Int("port", 8080, "http port")
var pVault = flag.String("vault", "localhost:8200", "vault address")

type vaultResponse struct {
	Errors []string `json:"errors,omitempty"`
	Data   struct {
		ID   string `json:"id,omitempty"`
		Meta struct {
			Permissions string `json:"permissions,omitempty"`
			Topic       string `json:"topic,omitempty"`
			Channels    string `json:"channels,omitempty"`
		} `json:"meta,omitempty"`
		DisplayName string `json:"display_name,omitempty"`
	} `json:"data,omitempty"`
}

type handler struct {
	vault string
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	query := r.URL.Query()
	token := query.Get("auth_secret")
	uri, err := url.Parse(h.vault)
	if err != nil {
		log.Printf("unablet to parse vault url: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	uri.Path = "/v1/auth/token/lookup-self"
	req, _ := http.NewRequest(http.MethodGet, uri.String(), nil)
	req = req.WithContext(r.Context())
	req.Header.Set("X-Vault-Token", token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	vaultRes := vaultResponse{}
	if err := json.Unmarshal(body, &vaultRes); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if len(vaultRes.Errors) > 0 {
		http.Error(w, strings.Join(vaultRes.Errors, "\n"), http.StatusUnauthorized)
		return
	}

	nsqRes := map[string]interface{}{
		"ttl":      3600,
		"identity": vaultRes.Data.DisplayName,
		"authorizations": []interface{}{
			map[string]interface{}{
				"permissions": strings.Split(vaultRes.Data.Meta.Permissions, ","),
				"topic":       vaultRes.Data.Meta.Topic,
				"channels":    strings.Split(vaultRes.Data.Meta.Channels, ","),
			},
		},
	}
	body, _ = json.Marshal(&nsqRes)
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func main() {
	flag.Parse()
	http.Handle("/auth", &handler{vault: *pVault})
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", *pHost, *pPort), nil); err != nil {
		log.Fatal(err)
	}
}
