package mail

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	mail "github.com/xhit/go-simple-mail/v2"
	"os"
	"strconv"
	"strings"
	"time"
	"time_speak_server/src/log"
)

type Svc struct {
	Config
	redis    *redis.Client
	template string
	server   *mail.SMTPServer
}

func NewMailSvc(conf Config, redis *redis.Client) *Svc {
	if conf.Template == "" {
		log.Fatal("fail to create mail svc: no email file")
	}
	template, err := os.ReadFile(conf.Template)
	if err != nil {
		log.Fatal("fail to create mail svc", "err", err)
	}
	s := conf
	server := mail.NewSMTPClient()
	// SMTP Server
	server.Host = s.SmtpMailHost
	server.Port = s.SmtpMailPort
	server.Username = s.SmtpMailUser
	server.Password = s.SmtpMailPwd
	server.Encryption = mail.EncryptionSSLTLS
	// Variable to keep alive connection
	server.KeepAlive = false
	// Timeout for connect to SMTP Server
	server.ConnectTimeout = 10 * time.Second
	// Timeout for send the data and wait respond
	server.SendTimeout = 10 * time.Second
	server.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err != nil {
		log.Fatal(err.Error())
	}

	return &Svc{
		Config:   conf,
		redis:    redis,
		template: string(template),
		server:   server,
	}
}

// NewEmailVerifyCode 创建邮箱验证码
func (s *Svc) NewEmailVerifyCode(ctx context.Context, email, act string) error {
	err := s.redis.Get(ctx, coolDownKey(email)).Err()
	if err != nil && err != redis.Nil {
		return err
	}
	if err == nil {
		return ErrVerifyCodeCoolDown
	}

	code := newCode(email, act, s.CodeLength)
	err = s.sendVerifyCode(email, code.Act, code.Code)
	if err != nil {
		return err
	}
	jsonStr, err := json.Marshal(code)
	if err != nil {
		return err
	}
	err = s.redis.Set(ctx, CodeKey(email), string(jsonStr), time.Duration(s.CodeExpire)*time.Minute).Err()
	if err != nil {
		return err
	}
	err = s.redis.Set(ctx, coolDownKey(email), true, time.Duration(s.CodeCoolDown)*time.Minute).Err()
	if err != nil {
		return err
	}

	return nil
}

func (s *Svc) sendMail(address string, subject, body string) (err error) {
	email := mail.NewMSG()
	email.SetFrom(fmt.Sprintf("%s<%s>", s.SmtpMailNickname, s.SmtpMailUser)).
		AddTo(address).
		SetSubject(subject)
	email.SetBody(mail.TextHTML, body)
	smtpClient, err := s.server.Connect()
	err = email.Send(smtpClient)
	if err != nil {
		return err
	}
	return
}

func (s *Svc) sendVerifyCode(address string, act string, code string) (err error) {
	body := strings.ReplaceAll(s.template, "${code}", code)
	body = strings.ReplaceAll(body, "${code_expire}", strconv.Itoa(s.CodeExpire))
	body = strings.ReplaceAll(body, "${act}", act)
	err = s.sendMail(address, s.Subject, body)
	return
}

func (s *Svc) CheckEmailVerifyCode(ctx context.Context, email, code string) bool {
	result, err := s.checkVerifyCode(ctx, email, code)
	if err != nil {
		log.Error("error in checking email code", "email", email, "err", err)
	}
	return result
}

func (s *Svc) checkVerifyCode(ctx context.Context, id, c string) (bool, error) {
	code := Code{ID: id}
	result, err := s.redis.Get(ctx, CodeKey(id)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	err = json.Unmarshal([]byte(result), &code)
	if err != nil {
		return false, err
	}
	return c == code.Code, nil
}
