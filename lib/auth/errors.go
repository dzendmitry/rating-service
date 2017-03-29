package auth

type ErrorUsersDoesntExist struct {}
func (e ErrorUsersDoesntExist) Error() string {
	return "User doesn't exist"
}

type ErrorUserExists struct {}
func (e ErrorUserExists) Error() string {
	return "User already exists"
}

type ErrorInvalidLogin struct {}
func (e ErrorInvalidLogin) Error() string {
	return "Invalid login"
}

type ErrorInvalidPassword struct {}
func (e ErrorInvalidPassword) Error() string {
	return "Invalid password"
}

type ErrorAlreadyAuthenticated struct {}
func (e ErrorAlreadyAuthenticated) Error() string {
	return "User already authenticated"
}

type ErrorNotAuthorized struct {}
func (e ErrorNotAuthorized) Error() string {
	return "User not authorized"
}