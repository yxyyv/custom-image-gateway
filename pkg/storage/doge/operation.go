package doge

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/haierkeys/custom-image-gateway/pkg/fileurl"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/pkg/errors"
)

func (d *Doge) GetBucket(bucketName string) string {
	if len(bucketName) <= 0 {
		bucketName = d.Config.BucketName
	}
	return bucketName
}

// SendFile 上传文件
func (d *Doge) SendFile(fileKey string, file io.Reader, itype string) (string, error) {
	ctx := context.Background()
	bucket := d.GetBucket("")

	fileKey = fileurl.PathSuffixCheckAdd(d.Config.CustomPath, "/") + fileKey

	client := d.getClient()
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(fileKey),
		Body:        file,
		ContentType: aws.String(itype),
	})

	if err != nil {
		return "", errors.Wrap(err, "doge")
	}

	return fileKey, nil
}

// SendContent 上传内容
func (d *Doge) SendContent(fileKey string, content []byte) (string, error) {
	ctx := context.Background()
	bucket := d.GetBucket("")

	fileKey = fileurl.PathSuffixCheckAdd(d.Config.CustomPath, "/") + fileKey

	client := d.getClient()
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileKey),
		Body:   bytes.NewReader(content),
	})

	if err != nil {
		return "", errors.Wrap(err, "doge")
	}

	return fileKey, nil
}

func (d *Doge) ObjectExists(fileKey string) (bool, error) {
	ctx := context.Background()
	bucket := d.GetBucket("")
	fileKey = fileurl.PathSuffixCheckAdd(d.Config.CustomPath, "/") + fileKey

	client := d.getClient()
	_, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileKey),
	})
	if err == nil {
		return true, nil
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound" {
		return false, nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "notfound") || strings.Contains(strings.ToLower(err.Error()), "404") {
		return false, nil
	}

	return false, errors.Wrap(err, "doge")
}
