package v1

// DatabaseSpec defines the specification for a database connection.
type DatabaseSpec struct {
	// URI is the uniform resource identifier for the database.
	URI string `json:"uri"`
	// Port is the network port on which the database is listening.
	Port int `json:"port"`
	// User is the username used to authenticate with the database.
	User string `json:"user"`
	// Pass is the password used to authenticate with the database.
	Pass string `json:"pass"`
	// DbName is the name of the database to connect to.
	DbName string `json:"dbName"`
	// DbType is the type of the database (e.g., postgresql, mysql).
	DbType string `json:"dbType"`
}

// S3Auth contains authentication information for accessing an S3-compatible storage service.
type S3Auth struct {
	// AccessKey is the access key ID used for authentication.
	AccessKey string `json:"accessKey"`
	// SecretKey is the secret access key used for authentication.
	SecretKey string `json:"secretKey"`
}

// S3Spec defines the specification for connecting to an S3-compatible storage service.
type S3Spec struct {
	Endpoint string `json:"endpoint"`
	// Auth contains the authentication information for accessing the S3 service.
	Auth S3Auth `json:"auth"`
	// BucketName is the name of the bucket to use within the S3 service.
	BucketName string `json:"bucketName"`
}
