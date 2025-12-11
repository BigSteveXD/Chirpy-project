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
	"errors"
	"sort"
	//"slices"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
	secret string
	polka string
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
    if err != nil {
		respondWithError(w, 500, "couldn't create chirp")
		return
	}

	respondWithJSON(w, 201, response{
		responseBody: responseBody{
			ID: myChirp.ID,
			CreatedAt: myChirp.CreatedAt,
			UpdatedAt: myChirp.UpdatedAt,
			Body: myChirp.Body,
			User_ID: myChirp.UserID,//myChirp.User_ID, //params.UserID, //userUUID
		},
    })
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
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		Password    string    `json:"-"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
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
		respondWithError(w, http.StatusInternalServerError, "couldn't hash password")
		return
	}
	
	params.Password = hashPass
	myUser, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{params.Password, params.Email})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't create user")
		return
	}

	//return users ID, email, and timestamps in response body
	respondWithJSON(w, http.StatusCreated, response{//201
		user: user{
			ID: myUser.ID,
			CreatedAt: myUser.CreatedAt,
			UpdatedAt: myUser.UpdatedAt,
			Email: myUser.Email,
			IsChirpyRed: myUser.IsChirpyRed,
		},
    })
}
func (cfg *apiConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	type email struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	type user struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		IsChirpyRed bool `json:"is_chirpy_red"`
	}
	type response struct {
		user
		Token     string    `json:"token"`
		RefreshToken string `json:"refresh_token"`
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

	oneUser, err := cfg.db.GetOneUser(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	passCheck, err := auth.CheckPasswordHash(params.Password, oneUser.HashedPassword)
	if passCheck {
		//create access token
		accessToken, err := auth.MakeJWT(oneUser.ID, cfg.secret, time.Hour)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "couldn't create access token")
			return
		}

		//create refresh token
		refreshToken, _ := auth.MakeRefreshToken()

		//create refresh token in database
		_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			refreshToken, time.Now(), oneUser.ID, time.Now().Add(60*60*24*60),//token, updated_at, user_id, expires_at, 
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "couldn't create refresh in the database")
		}

		respondWithJSON(w, http.StatusOK, response{//201
		user: user{
			ID: oneUser.ID,
			CreatedAt: oneUser.CreatedAt,
			UpdatedAt: oneUser.UpdatedAt,
			Email: oneUser.Email,
			//revoke is null
			IsChirpyRed: oneUser.IsChirpyRed,
		},
		Token: accessToken,
		RefreshToken: refreshToken,
    	})
	}else{
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")//401
		return
	}
}
func (cfg *apiConfig) handleRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token     string    `json:"token"`//refresh
	}
	//get refresh token from headers in the same format as "Authorization: Bearer <token>"
	tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couldn't find JWT")//401
		return
	}
	//use token to get user from database
	oneUser, err := cfg.db.GetUserFromRefreshToken(r.Context(), tokenStr)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "couldn't get user from refresh token")//401
		return
	}
	//check if refresh token is expired
	if oneUser.ExpiresAt.After(time.Now()) {
		respondWithError(w, http.StatusInternalServerError, "token is expired")
		return
	}
	//check if refresh token is revoked
	revokeVal, err := oneUser.RevokedAt.Value()//Valid, Time
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't get RevokedAt Value()")
		return
	}
	if revokeVal == nil {
		//create new access token for the user
		accessToken, err := auth.MakeJWT(oneUser.UserID, cfg.secret, time.Hour)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "couldn't create new access token")
			return
		}

		respondWithJSON(w, 200, response{
			Token: accessToken,
		})
	}else{
		respondWithError(w, 401, "token is revoked")
		return
	}
}
func (cfg *apiConfig) handleRevoke(w http.ResponseWriter, r *http.Request) {
	tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "couldn't find JWT")//401
		return
	}

	//arg.UpdatedAt, arg.RevokedAt, arg.Token
	cfg.db.UpdateRefreshToken(r.Context(), database.UpdateRefreshTokenParams{time.Now(), sql.NullTime{time.Now(), true}, tokenStr})

	w.WriteHeader(http.StatusNoContent)//204
}

type response struct {
	responseBody
}
func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request, author_id string, mySort string) {
	var outputs []response

	if mySort == "" {
		mySort = "asc"
	}

	if author_id == "" {
		//get all chirps
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
	}else{
		//get user's chirps
		userUUID, err := uuid.Parse(author_id)
		if err != nil {
			respondWithError(w, 404, "failed to parse uuid from string")
			return
		}
		userChirps, err := cfg.db.GetUserChirps(r.Context(), userUUID)
		if err != nil {
			respondWithError(w, 500, "failed to get chirps")//500 server error response
		}

		for _, chirp := range userChirps{
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
	}
	
	if mySort == "desc" {
		//slices.Reverse(outputs)
		sort.Slice(outputs, func(i, j int) bool {
			return outputs[i].CreatedAt.After(outputs[j].CreatedAt)
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

func (cfg *apiConfig) putUsers(w http.ResponseWriter, r *http.Request) {
	//get access token
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

	//get email and password
	type email struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	defer r.Body.Close()
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

	hashPass, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't hash password")
		return
	}

	//update password and email for authenticated user in database
	type user struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		IsChirpyRed bool `json:"is_chirpy_red"`
	}
	type response struct {
		user
	}
	//updated_at, email, hashed_password, id
	cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{time.Now(), params.Email, hashPass, userUUID})

	oneUser, err := cfg.db.GetOneUser(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	respondWithJSON(w, 200, response{
		user: user{
			ID: oneUser.ID,
			CreatedAt: oneUser.CreatedAt,
			UpdatedAt: oneUser.UpdatedAt,
			Email: oneUser.Email,
			IsChirpyRed: oneUser.IsChirpyRed,
		},
    })
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request, chirpID string) {
	//authenticate token
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

	//parse chirp id from string
	temp, err := uuid.Parse(chirpID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "failed to parse uuid from string")
		return
	}
	//check if chirp exists
	oneChirp, err := cfg.db.GetOneChirp(r.Context(), temp)
	if err != nil {
		respondWithError(w, 404, "failed to get chirp")
		return
	}

	//check if user is author of chirp
	if oneChirp.UserID != userUUID {
		respondWithError(w, 403, "author of chirp and user don't match")
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), oneChirp.ID)
	if err != nil {
		respondWithError(w, 400, "couldn't delete chirp")
		return
	}
	w.WriteHeader(http.StatusNoContent)//204
}

func (cfg *apiConfig) handleWebhook(w http.ResponseWriter, r *http.Request) {
	//Polka uses the response code to know whether or not the webhook was received successfully. 
	//If the response code is anything other than 2XX, they'll retry the request.
	type responseBody struct {
		UserID        uuid.UUID `json:"user_id"`
	}
	type response struct {
		Event        string `json:"event"`//bool
		Data         responseBody `json:"data"`
	}

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "couldn't find API key")//401
		return
	}

	if apiKey != cfg.polka {
		respondWithError(w, 401, "api key doesn't match")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := response{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't decode parameters")
		return
	}

	if params.Event != "user.upgraded" {
		respondWithError(w, 204, "user is not upgraded")
		return
	}

	err = cfg.db.UpgradeUser(r.Context(), params.Data.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "user can't be found")//http.StatusNotFound
			return
		}
		respondWithError(w, http.StatusInternalServerError, "couldn't update user")
		return
	}

	w.WriteHeader(http.StatusNoContent)//204
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
	polk := os.Getenv("POLKA_KEY")

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db: dbQueries,
		platform: plat,
		secret: secr,
		polka: polk,
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

	myServeMux.Handle("POST /api/refresh", http.HandlerFunc(apiCfg.handleRefresh))
	myServeMux.Handle("POST /api/revoke", http.HandlerFunc(apiCfg.handleRevoke))

	myServeMux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request){
		s := r.URL.Query().Get("author_id")
		//s is a string that contains the value of the author_id query parameter
		//if it exists, or an empty string if it doesn't
		sort := r.URL.Query().Get("sort")
		apiCfg.getChirps(w, r, s, sort)
	})
	myServeMux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request){
		id := r.PathValue("chirpID")
		apiCfg.getChirp(w, r, id)
	})

	myServeMux.HandleFunc("PUT /api/users", http.HandlerFunc(apiCfg.putUsers))

	myServeMux.HandleFunc("DELETE /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request){
		id := r.PathValue("chirpID")
		apiCfg.deleteChirp(w, r, id)
	})

	myServeMux.Handle("POST /api/polka/webhooks", http.HandlerFunc(apiCfg.handleWebhook))

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

