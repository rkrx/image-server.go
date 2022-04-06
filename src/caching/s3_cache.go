package caching

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"log"
	fp "path/filepath"
)

type S3Config struct {
	Key        string
	Secret     string
	Endpoint   string
	Bucket     string
	PathPrefix string
	Region     string
	PublicAddr string
}

func (conf *S3Config) NewSession() (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(conf.Key, conf.Secret, ""),
		Endpoint:    aws.String(conf.Endpoint),
		Region:      aws.String(conf.Region),
	})
}

func (conf *S3Config) concatFilename(filepath string) string {
	return fp.Join(conf.PathPrefix, filepath)
}

func (conf *S3Config) Has(filepath string) (bool, error) {
	sess, err := conf.NewSession()
	if err != nil {
		return false, err
	}
	s3svc := s3.New(sess)
	_, err = s3svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(conf.Bucket),
		Key:    aws.String(conf.concatFilename(filepath)),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NotFound": // s3.ErrCodeNoSuchKey does not work, aws is missing this error code so we hardwire a string
				return false, nil
			default:
				log.Printf("Code: %s", aerr.Code())
				return false, err
			}
		}
	}
	return true, nil
}

func (conf *S3Config) Get(filepath string, buffer io.Writer) (*s3.GetObjectOutput, error) {
	sess, err := conf.NewSession()
	if err != nil {
		return nil, err
	}
	s3svc := s3.New(sess)
	result, err := s3svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(conf.Bucket),
		Key:    aws.String(conf.concatFilename(filepath)),
	})
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(buffer, result.Body)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (conf *S3Config) Put(filepath, contentType string, reader io.ReadSeeker) error {
	sess, err := conf.NewSession()
	if err != nil {
		return err
	}
	s3svc := s3.New(sess)

	_, err = s3svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(conf.Bucket),
		Key:         aws.String(conf.concatFilename(filepath)),
		Body:        reader,
		ACL:         aws.String("public-read"),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return aerr
		}
	}
	return nil
}
