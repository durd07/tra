package main

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

type SkeyResponse struct {
	Skey string `json:"skey"`
	Nodes []struct {
		Nodeid int `json:"nodeid"`
		Ip string `json:"ip"`
	} `json:"nodes"`
}

const SKEY_URL = "http://localhost:15810/SIP/INT/nodes?skey="

func query_skey(skey string) string {
	resp, err := http.Get(SKEY_URL + skey)
	if err != nil {
		fmt.Println("No response for request", err)
		return ""
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	var skey_resps []SkeyResponse
	if err := json.Unmarshal(body, &skey_resps); err != nil {
		fmt.Println("Can not unmarshal JSON")
	}

	return skey_resps[0].Nodes[0].Ip
}
