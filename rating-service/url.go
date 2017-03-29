package main

import (
	"github.com/dzendmitry/rating-service/lib/general"
)

const (
	ADD_URL = "add"
	FIND_URL = "find"
	EDIT_URL = "edit"
	GET_URL = "get"
	REMOVE_URL = "remove"
)

func getAllContentUrl() string {
	return general.BASE_URL_V1 + GET_URL
}

func movieUrl() string {
	return general.BASE_URL_V1 + general.TYPE_MOVIE
}

func getMoviesUrl() string {
	return movieUrl() + "/" + GET_URL
}

func addMovieUrl() string {
	return movieUrl() + "/" + ADD_URL
}

func findMovieUrl() string {
	return movieUrl() + "/" + FIND_URL
}

func editMovieUrl() string {
	return movieUrl() + "/" + EDIT_URL
}

func removeMovieUrl() string {
	return movieUrl() + "/" + REMOVE_URL
}

func bookUrl() string {
	return general.BASE_URL_V1 + general.TYPE_BOOK
}

func getBooksUrl() string {
	return bookUrl() + "/" + GET_URL
}

func addBookUrl() string {
	return bookUrl() + "/" + ADD_URL
}

func findBookUrl() string {
	return bookUrl() + "/" + FIND_URL
}

func editBookUrl() string {
	return bookUrl() + "/" + EDIT_URL
}

func removeBookUrl() string {
	return bookUrl() + "/" + REMOVE_URL
}