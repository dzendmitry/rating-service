package general

const (
	BASE_URL_V1 = "/api/v1/"

	FIND_BY_NAME   = "byName"
	FIND_BY_YEAR   = "byYear"
	FIND_BY_ISBN   = "byISBN"
	FIND_BY_ACTOR  = "byActor"
	FIND_BY_WRITER = "byWriter"
	FIND_BY_PERIOD = "byPeriod"

	BODY_BUFFER = 1 * 1024 * 1024
	ONE_PARSER_REQUEST_TIMEOUT = 2
	ONE_REQUEST_TIMEOUT = 500
)
