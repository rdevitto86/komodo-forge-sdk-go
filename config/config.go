package config

const (
	// --- server/docker environment variables ---------------------------------------------

	APP_NAME     = "APP_NAME"
	LOG_LEVEL    = "LOG_LEVEL"
	ENV          = "ENV"
	PORT         = "PORT"         // public port
	PORT_PRIVATE = "PORT_PRIVATE" // internal/private port
	PORT_METRICS = "PORT_METRICS"

	// --- aws environment variables ---------------------------------------------

	AWS_REGION        = "AWS_REGION"
	AWS_ENDPOINT      = "AWS_ENDPOINT"
	AWS_SECRET_PREFIX = "AWS_SECRET_PREFIX"
	AWS_SECRET_BATCH  = "AWS_SECRET_BATCH"

	DYNAMODB_ENDPOINT   = "DYNAMODB_ENDPOINT"
	DYNAMODB_TABLE      = "DYNAMODB_TABLE"
	DYNAMODB_ACCESS_KEY = "DYNAMODB_ACCESS_KEY"
	DYNAMODB_SECRET_KEY = "DYNAMODB_SECRET_KEY"

	// --- http environment variables ---------------------------------------------

	HOST                = "HOST"
	MAX_CONTENT_LENGTH  = "MAX_CONTENT_LENGTH"
	RATE_LIMIT_RPS      = "RATE_LIMIT_RPS"
	RATE_LIMIT_BURST    = "RATE_LIMIT_BURST"
	BUCKET_TTL_SECOND   = "BUCKET_TTL_SECOND"
	IP_WHITELIST        = "IP_WHITELIST"
	IP_BLACKLIST        = "IP_BLACKLIST"
	IDEMPOTENCY_TTL_SEC = "IDEMPOTENCY_TTL_SEC"

	// --- security environment variables ---------------------------------------------

	JWT_PUBLIC_KEY  = "JWT_PUBLIC_KEY"
	JWT_PRIVATE_KEY = "JWT_PRIVATE_KEY"
	JWT_AUDIENCE    = "JWT_AUDIENCE"
	JWT_ISSUER      = "JWT_ISSUER"
	JWT_KID         = "JWT_KID"
)
