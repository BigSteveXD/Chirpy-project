package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	//"io"
	"bytes"
	//"log"
)

type parameters struct {//use for request //can also use for response or use another struct
	Body string `json:"body"`
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
type emailBody struct {
	Email string `json:"email"`
}
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func main(){
	url := "http://localhost:8080/api/validate_chirp"

	//myRequest := requestBody{
		//Body:`This is a kerfuffle opinion I need to share with the world`,
	//}
	myRequest := emailBody{
		Email: "mloneusk@example.co"
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

	//var respData struct {
		//Body string `json:"body"`
	//}
	//var temp responseBody
	temp := User{}//responseBody{}
	err = json.NewDecoder(resp.Body).Decode(&temp)//&parameters//&respData
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println(resp)//resp.Body
	//fmt.Println(resp.Body)//resp.Body
	//fmt.Println(resp.Body)
	//fmt.Println(temp.Cleaned_body)
	
	fmt.Println(temp.ID)
	fmt.Println(temp.CreatedAt)
	fmt.Println(temp.UpdatedAt)
	fmt.Println(temp.Email)
}
