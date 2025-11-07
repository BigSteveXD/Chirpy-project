package main

import (
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"os"
	"database/sql"
	"github.com/BigSteveXD/Chirpy-project/internal/database"
	"github.com/BigSteveXD/Chirpy-project/internal/auth"
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
	secret string
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
	UserID uuid.UUID `json:"user_id"`//User_ID //Authorization
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
    params := requestBody{}
    err = json.Unmarshal(dat, &params)
    if err != nil {
        respondWithError(w, 500, "couldn't unmarshal parameters")
        return
    }
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	bearerString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "couldn't find JWT")//401
		return
	}
	
	userUUID, err := auth.ValidateJWT(bearerString, cfg.secret)//tokenString, tokenSecret
	if err != nil {
		respondWithError(w, 401, "couldn't validate JWT")
		return
	}
	

	cleaned_body := replaceBadWords(params)
	//fmt.Println(cleaned_body)
	params.Body = cleaned_body

	//create chirp in database
	type response struct {
		responseBody
	}
	myChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{params.Body, userUUID})//sql.NullString //params.Body, params.UserID
    
	err = respondWithJSON(w, 201, response{
		responseBody: responseBody{
			ID: myChirp.ID,
			CreatedAt: myChirp.CreatedAt,
			UpdatedAt: myChirp.UpdatedAt,
			Body: myChirp.Body,
			User_ID: myChirp.UserID,//myChirp.User_ID, //params.UserID, //userUUID
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
		Password string `json:"password"`
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
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't unmarshal params")
		return
	}

	//hash password then create user
	hashPass, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't respond with json")
		return
	}
	
	params.Password = hashPass
	myUser, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{params.Password, params.Email})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't respond with json")
	}

	//return users ID, email, and timestamps in response body
	respondWithJSON(w, http.StatusCreated, response{//201
		user: user{
			ID: myUser.ID,
			CreatedAt: myUser.CreatedAt,
			UpdatedAt: myUser.UpdatedAt,
			Email: myUser.Email,
		},
    })
}
func (cfg *apiConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	type email struct {
		Password string `json:"password"`
		Email string `json:"email"`
		ExpiresInSeconds time.Duration `json:"expires_in_seconds"`//optional //int
	}
	type user struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		Token     string    `json:"token"`
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
	if params.ExpiresInSeconds != 0 {
		if params.ExpiresInSeconds > time.Hour {//1hour
			params.ExpiresInSeconds = time.Hour
		}
	}else{
		params.ExpiresInSeconds = time.Hour
	}

	oneUser, err := cfg.db.GetOneUser(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	passCheck, err := auth.CheckPasswordHash(params.Password, oneUser.HashedPassword)
	if passCheck {
		//create token
		token, err := auth.MakeJWT(oneUser.ID, cfg.secret, params.ExpiresInSeconds)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "couldn't create token")
			return
		}

		respondWithJSON(w, http.StatusOK, response{//201
		user: user{
			ID: oneUser.ID,
			CreatedAt: oneUser.CreatedAt,
			UpdatedAt: oneUser.UpdatedAt,
			Email: oneUser.Email,
			Token: token,
		},
    	})
	}else{
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")//401
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
func (cfg *apiConfig) getChirp(w http.ResponseWriter, r *http.Request, chirpID string) {
	temp, err := uuid.Parse(chirpID)
	if err != nil {
		respondWithError(w, 404, "failed to parse uuid from string")
		return
	}
	oneChirp, err := cfg.db.GetOneChirp(r.Context(), temp)
	if err != nil {
		respondWithError(w, 404, "failed to get chirp")
		return
	}
	respondWithJSON(w, 200, response{
		responseBody: responseBody{ 
			ID: oneChirp.ID,
			CreatedAt: oneChirp.CreatedAt,
			UpdatedAt: oneChirp.UpdatedAt,
			Body: oneChirp.Body,
			User_ID: oneChirp.UserID,
		},
    })
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
	secr := os.Getenv("JWT_SECRET")

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db: dbQueries,
		platform: plat,
		secret: secr,
	}

	myServeMux := http.NewServeMux()

	myServeMux.Handle("/app/", apiCfg.middlewareMetricsInc( http.StripPrefix("/app", http.FileServer(http.Dir("."))) ))

	myServeMux.Handle("GET /admin/metrics", http.HandlerFunc(apiCfg.countHits))
	myServeMux.Handle("POST /admin/reset", http.HandlerFunc(apiCfg.resetHits))

	myServeMux.Handle("POST /api/chirps", http.HandlerFunc(apiCfg.handleChirps))

	myServeMux.Handle("POST /api/users", http.HandlerFunc(apiCfg.handleUsers))
	myServeMux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request){
		apiCfg.handleLogin(w, r)
	})

	myServeMux.Handle("GET /api/chirps", http.HandlerFunc(apiCfg.getChirps))
	myServeMux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request){
		id := r.PathValue("chirpID")
		apiCfg.getChirp(w, r, id)
	})

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

