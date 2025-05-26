package v1

type DatabaseSpec struct {
	URI    string `json:"uri"`
	Port   int    `json:"port"`
	User   string `json:"user"`
	Pass   string `json:"pass"`
	DbName string `json:"dbName"`
	DbType string `json:"dbType"`
}

type S3Auth struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}
type S3Spec struct {
	Endpoint   string `json:"endpoint"`
	Auth       S3Auth `json:"auth"`
	BucketName string `json:"bucketName"`
}
