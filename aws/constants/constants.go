package constants

// AWS US region codes. Use these when constructing client Configs so region
// strings are consistent across services and a typo surfaces as a compile error
// instead of a runtime "unknown region" from the AWS SDK.
const (
	AWS_US_EAST_1 = "us-east-1"
	AWS_US_EAST_2 = "us-east-2"
	AWS_US_WEST_1 = "us-west-1"
	AWS_US_WEST_2 = "us-west-2"
)
