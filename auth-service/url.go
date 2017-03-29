package main

import "github.com/dzendmitry/rating-service/lib/general"

const (
	AUTH  = "authenticate"
	REG   = "register"
	UNREG = "unregister"
	EXIT  = "exit"
)

func authUrl() string {
	return general.BASE_URL_V1 + AUTH
}

func exitUrl() string {
	return general.BASE_URL_V1 + EXIT
}

func regUrl() string {
	return general.BASE_URL_V1 + REG
}

func unregUrl() string {
	return general.BASE_URL_V1 + UNREG
}