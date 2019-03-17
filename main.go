package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/renavides/prueba/client"
	"github.com/renavides/prueba/config"
	"log"
	"github.com/dimiro1/health"
	"github.com/dimiro1/health/url"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Starting server initialization")
	env, ok := os.LookupEnv("ENV")
	if ok != true {
		log.Printf("Environment: %s \n", env)
		env = "dev"
	}else{
		log.Printf("Environment: %s \n", env)
	}
	var config = config.Config{}
	config.Read()

	//Server params
	var credential = client.Credential{
		Token:          config.Vault.Credential.Token,
		RoleID:         config.Vault.Credential.RoleID,
		SecretID:       config.Vault.Credential.SecretID,
		ServiceAccount: config.Vault.Credential.ServiceAccount,
	}

	var vault = client.Vault{
		Host:           config.Vault.Host,
		Port:           config.Vault.Port,
		Scheme:         config.Vault.Scheme,
		Authentication: config.Vault.Authentication,
		Role:           config.Vault.Role,
		Mount:          config.Vault.Mount,
		Credential:     credential,
	}

	//Init it
	log.Println("Starting vault initialization")
	err := vault.Initialize()
	if err != nil {
		log.Fatal(err)
	}

	//Router
	r := mux.NewRouter()

	//API Routes
	r.HandleFunc("/api/secret", func (w http.ResponseWriter, r *http.Request) {
		prm := mux.Vars(r)
		fmt.Println(prm)
		secret, err := vault.GetSecret("secret-v1/qms-views/dev")
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if secret.Data != nil {
			respondWithJson(w, http.StatusOK, secret)
		} else {
			respondWithJson(w, http.StatusOK, map[string]string{"result": "No secret"})
		}
	}).Methods("GET")

	r.HandleFunc("/api/secret/{service}", func (w http.ResponseWriter, r *http.Request) {
		prm := mux.Vars(r)
		secret, err := vault.GetSecret(fmt.Sprintf("secret-v1/%s/%s", prm["service"], env))
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if secret.Data != nil {
			respondWithJson(w, http.StatusOK, secret)
		} else {
			respondWithJson(w, http.StatusOK, map[string]string{"result": "No secret"})
		}
	}).Methods("GET")



	//Health Check Routes
	h := health.NewHandler()
	h.AddChecker("Vault", url.NewChecker(fmt.Sprintf("%s://%s:%s/v1/sys/health?perfstandbyok=true", config.Vault.Scheme, config.Vault.Host, config.Vault.Port)))
	r.Path("/health").Handler(h).Methods("GET")

	//Server config - http
	go func() {
		log.Println(fmt.Sprintf("Server is now accepting http requests on port %v", config.Server.Port))
		if err := http.ListenAndServe(fmt.Sprintf(":%v", config.Server.Port), r); err != nil {
			log.Fatal(err)
		}
	}()

	//Catch SIGINT AND SIGTERM to gracefully tear down tokens and secrets
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	sig := <-gracefulStop
	fmt.Printf("caught sig: %+v", sig)
	vault.Close()
	os.Exit(0)

}


func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJson(w, code, map[string]string{"error": msg})
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}