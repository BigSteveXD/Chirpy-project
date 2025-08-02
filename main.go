package main

import (
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"os"
	"database/sql"
	"github.com/BigSteveXD/Chirpy-project/internal/database"
	"time"
	"github.com/google/uuid"
	"log"
	"net/http"
	"sync/atomic"
	"fmt"
    "encoding/json"
    "io"
	"strings"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	myQueries *database.Queries
	platform string
}
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
func (cfg *apiConfig) countHits(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")//text/plain
	w.WriteHeader(http.StatusOK)
	//fmt.Fprintf(w, "Hits: %d", hits)
	//w.Write([]byte(fmt.Sprintf("Hits: %d", hits)))
	fmt.Fprintf(w, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, hits)
}
func (cfg *apiConfig) resetHits(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, 403, "Forbidden")
		return
	}

	cfg.fileserverHits.Store(0)

	//delete all users in database(not schema)
	err := cfg.myQueries.Reset(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("reset failed: " + err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("database reset"))
}

type requestBody struct {
	Body string `json:"body"`
}
type responseBody struct {
	Cleaned_body string `json:"cleaned_body"`//Body string `json:"body"`
}
type validBody struct {
	Valid bool `json:"valid"`
}
func handleHTTP(w http.ResponseWriter, r *http.Request){
    defer r.Body.Close()
	
    dat, err := io.ReadAll(r.Body)
    if err != nil {
        respondWithError(w, 500, "couldn't read request")
        return
    }
	if len(dat) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
    params := requestBody{}
    err = json.Unmarshal(dat, &params)
    if err != nil {
        respondWithError(w, 500, "couldn't unmarshal parameters")
        return
    }

	cleaned_body := replaceBadWords(params)
	fmt.Println(cleaned_body)
    
	err = respondWithJSON(w, 200, responseBody{
		Cleaned_body: cleaned_body,
    })
	
	if err != nil {
		respondWithError(w, 500, "couldn't respond with json")
		return
	}
}
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
    response, err := json.Marshal(payload)
    if err != nil {
		respondWithError(w, 500, "couldn't marshal payload")
		return nil
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(response)
    return nil
}
func respondWithError(w http.ResponseWriter, code int, msg string) error {
    //return respondWithJSON(w, code, map[string]string{"error": msg})
	return respondWithJSON(w, code, struct{Error string `json:"error"`}{Error:msg})
}
func replaceBadWords(words interface{}) string {
	temp := strings.Split(words.(requestBody).Body, " ")
	for x := range(len(temp)){
		if strings.ToLower(temp[x]) == "kerfuffle" || 
		strings.ToLower(temp[x]) == "sharbert" || 
		strings.ToLower(temp[x]) == "fornax" {
			temp[x] = "****"
		}
	}
	cleaned := strings.Join(temp, " ")
	return cleaned
}


func (cfg *apiConfig) handleUsers(w http.ResponseWriter, r *http.Request) {
	type email struct {
		Email string `json:"email"`
	}
	type user struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}
	defer r.Body.Close()
	//accept email as json in request body
	dat, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, 500, "couldn't read request")
	}
	params := email{}
	err = json.Unmarshal(dat, &params)

	//create user
	//user, err := cfg.db.CreateUser(r.Context(), params.Email)
	myUser, err := cfg.myQueries.CreateUser(r.Context(), params.Email)

	//return users ID email timestamps in response body
	//respondWithJSON(w, 201, "Created")
	err = respondWithJSON(w, 201, user{
		ID: myUser.ID,
		CreatedAt: myUser.CreatedAt,
		UpdatedAt: myUser.UpdatedAt,
		Email: params.Email,//myUser.Email
    })
	
	if err != nil {
		respondWithError(w, 500, "couldn't respond with json")
		return
	}
}


func main() {
	godotenv.Load()//if empty default loads .env from current path
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
	}
	dbQueries := database.New(db)

	plat := os.Getenv("PLATFORM")

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		myQueries: dbQueries,
		platform: plat,
	}

	myServeMux := http.NewServeMux()

	myServeMux.Handle("/app/", apiCfg.middlewareMetricsInc( http.StripPrefix("/app", http.FileServer(http.Dir("."))) ))//fileserver is a handler

	myServeMux.Handle("GET /admin/metrics", http.HandlerFunc(apiCfg.countHits))
	myServeMux.Handle("POST /admin/reset", http.HandlerFunc(apiCfg.resetHits))

	myServeMux.Handle("POST /api/validate_chirp", http.HandlerFunc(handleHTTP))

	myServeMux.Handle("POST /api/users", http.HandlerFunc(apiCfg.handleUsers))

	//readiness endpoint
	myServeMux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	//custom server
	myServer := &http.Server{
		Addr: ":8080",
		Handler: myServeMux,
	}
	log.Fatal(myServer.ListenAndServe())
}
