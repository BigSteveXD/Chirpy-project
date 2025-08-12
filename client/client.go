package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	//"io"
	"bytes"
	//"log"
	"time"
	"github.com/google/uuid"
)

type requestBody struct {//parameters
	Body string `json:"body"`
}
type responseBody struct {
	Cleaned_body string `json:"cleaned_body"`//Body string `json:"body"`
}
type validBody struct {
	Valid bool `json:"valid"`
}
type emailBody struct {
	Email string `json:"email"`
}
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}
type userRequest struct {//requestBody
	Body string `json:"body"`
	UserID uuid.UUID `json:"user_id"`//User_ID
}
type userResponse struct {//responseBody
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body string `json:"body"`
	User_ID uuid.UUID `json:"user_id"`
}

func main(){
	url := "http://localhost:8080/api/validate_chirp"

	/*
	myRequest := requestBody{
		Body:`This is a kerfuffle opinion I need to share with the world`,
	}
	*/
	/*
	myRequest := emailBody{
		Email: "mloneusk@example.co",
	}
	*/
	myRequest := userRequest{
		Body: "If you're committed enough, you can make any story work.",
		UserID: uuid.New(),//uuid.NewString(),//"123e4567-e89b-12d3-a456-426614174000",//`${userID1}`,
	}

	buff := new(bytes.Buffer)
	err := json.NewEncoder(buff).Encode(myRequest)
	if err != nil {
		fmt.Println(err)
		return
	}

	resp, err := http.Post(url, "application/json", buff)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	/*
	var respData struct {
		Body string `json:"body"`
	}
	*/
	//var temp responseBody
	//temp := User{}//responseBody{}
	/*
	type response struct{
		User
	}
	*/
	type response struct{
		userResponse
	}

	//
	temp := response{}//response
	err = json.NewDecoder(resp.Body).Decode(&temp)//&parameters//&respData
	if err != nil {
		fmt.Println(err)
		return
	}
	//
	//var temp userResponse = json.Unmarshal([]byte(resp.Body), &temp)

	//fmt.Println(resp)//resp.Body
	//fmt.Println(resp.Body)//resp.Body
	//fmt.Println(resp.Body)
	//fmt.Println(temp.Cleaned_body)
	/*
	fmt.Println(temp.User.ID)
	fmt.Println(temp.User.CreatedAt)
	fmt.Println(temp.User.UpdatedAt)
	fmt.Println(temp.User.Email)
	*/
	
	//fmt.Println(temp.userResponse.User_ID)\
	//fmt.Println(temp.User_ID)
}
