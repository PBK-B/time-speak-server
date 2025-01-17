package bce

type Config struct {
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	EndPoint        string `yaml:"end_point"`
	BucketName      string `yaml:"bucket_name"`
	Region          string `yaml:"region"`
	CDN             bool   `yaml:"cdn"`
	CdnAuthKey      string `yaml:"cdn_auth_key"`
	CallbackToken   string `yaml:"callback_token"`
}
