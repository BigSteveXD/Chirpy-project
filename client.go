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
type validBody struct {
	Valid bool `json:"valid"`
}

func main(){
	url := "http://localhost:8080/api/validate_chirp"//http://localhost:8080//api/validate_chirp

	myRequest := parameters{
		//Body:`body:"my request"`,
		Body:`my request`,
	}

	/*
	jsonData, err := json.Marshal(myRequest)
	if err != nil {
		fmt.Println(err)
		return
	}

	resp, err := http.Post(url, "application/json",  bytes.NewBuffer(jsonData))
	if err != nil{
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	//
	body, err := io.ReadAll(resp.Body)
	if err != nil{
		fmt.Println(err)
		return
	}

	//var responseData parameters
	var responseData validBody
	err = json.Unmarshal([]byte(body), &responseData)//body //respData

	if err != nil{
		fmt.Println(err)
		return
	}
	*/


	//buff := &bytes.Buffer{}//NewBuffer(buff []byte)
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

	//var validTest struct {
	//	Valid bool `json:"valid"`
	//}
	validTest := validBody{
		Valid:true,
	}
	err = json.NewDecoder(resp.Body).Decode(&validTest)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println(responseData.Body)
	//fmt.Println(responseData.Valid)
	fmt.Println(resp.Body)
}
