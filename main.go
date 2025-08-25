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
	db *database.Queries
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
	err := cfg.db.Reset(r.Context())
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
	UserID uuid.UUID `json:"user_id"`//User_ID
}
type responseBody struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body string `json:"body"`
	User_ID uuid.UUID `json:"user_id"`
}
func (cfg *apiConfig) handleChirps(w http.ResponseWriter, r *http.Request){
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
	//fmt.Println(cleaned_body)
	params.Body = cleaned_body

	//create chirp in database
	type response struct {
		responseBody
	}
	myChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{params.Body, params.UserID})//sql.NullString
    
	err = respondWithJSON(w, 201, response{
		responseBody: responseBody{
			ID: myChirp.ID,
			CreatedAt: myChirp.CreatedAt,
			UpdatedAt: myChirp.UpdatedAt,
			Body: myChirp.Body,
			User_ID: params.UserID,//myChirp.User_ID,
		},
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
	type response struct {
		user
	}
	defer r.Body.Close()
	//accept email as json in request body
	dat, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, 500, "couldn't read request")
		return
	}
	params := email{}
	err = json.Unmarshal(dat, &params)

	//create user
	myUser, err := cfg.db.CreateUser(r.Context(), params.Email)

	//return users ID email timestamps in response body
	//respondWithJSON(w, 201, "Created")
	err = respondWithJSON(w, 201, response{
		user: user{
			ID: myUser.ID,
			CreatedAt: myUser.CreatedAt,
			UpdatedAt: myUser.UpdatedAt,
			Email: myUser.Email,//params.Email,
		},
    })
	
	if err != nil {
		respondWithError(w, 500, "couldn't respond with json")
		return
	}
}

type response struct {
		responseBody
	}
func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	var outputs []response

	allChirps, err := cfg.db.GetAllChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, "failed to get chirps")//500 server error response
	}
	for _, chirp := range allChirps{
		outputs = append(outputs, response{
			responseBody: responseBody{ 
				ID: chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body: chirp.Body,
				User_ID: chirp.UserID,
			},
    	})
	}

	temp, err := json.Marshal(outputs)
	if err != nil {
		respondWithError(w, 500, "couldn't marshal payload")
    }
	w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200)//code
	w.Write([]byte(temp))
}


func main() {
	godotenv.Load()//if empty default loads .env from current path
	dbURL := os.Getenv("DB_URL")
	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
	}
	dbQueries := database.New(dbConn)

	plat := os.Getenv("PLATFORM")

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db: dbQueries,
		platform: plat,
	}

	myServeMux := http.NewServeMux()

	myServeMux.Handle("/app/", apiCfg.middlewareMetricsInc( http.StripPrefix("/app", http.FileServer(http.Dir("."))) ))

	myServeMux.Handle("GET /admin/metrics", http.HandlerFunc(apiCfg.countHits))
	myServeMux.Handle("POST /admin/reset", http.HandlerFunc(apiCfg.resetHits))

	myServeMux.Handle("POST /api/chirps", http.HandlerFunc(apiCfg.handleChirps))

	myServeMux.Handle("POST /api/users", http.HandlerFunc(apiCfg.handleUsers))

	myServeMux.Handle("GET /api/chirps", http.HandlerFunc(apiCfg.getChirps))

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

