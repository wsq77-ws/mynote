package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// OSSStorage 对象存储（兼容 S3 协议）
// 支持 AWS S3、MinIO、阿里云 OSS、腾讯云 COS 等
type OSSStorage struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewOSSStorage 创建对象存储实例
func NewOSSStorage(cfg OSSConfig) (*OSSStorage, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("对象存储 endpoint 不能为空")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("对象存储 bucket 不能为空")
	}

	// 构建 AWS 配置
	awsCfg, err := awscfg.LoadDefaultConfig(context.TODO(),
		awscfg.WithRegion(cfg.Region),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey, cfg.SecretKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("创建对象存储配置失败: %w", err)
	}

	// 创建 S3 客户端，指向自定义 endpoint
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true // MinIO 等需要 path-style
	})

	// 规范化前缀，确保以 / 结尾（如果非空）
	prefix := cfg.Prefix
	if prefix != "" {
		prefix = strings.TrimPrefix(prefix, "/")
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
	}

	return &OSSStorage{
		client: client,
		bucket: cfg.Bucket,
		prefix: prefix,
	}, nil
}

// fullKey 将相对路径转为完整的对象 key
func (s *OSSStorage) fullKey(path string) string {
	path = strings.TrimPrefix(path, "/")
	return s.prefix + path
}

// List 列出指定目录下的条目（非递归）
func (s *OSSStorage) List(dirPath string) ([]Entry, error) {
	// 构建前缀
	prefix := s.fullKey(dirPath)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	result, err := s.client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("列出对象失败: %w", err)
	}

	var entries []Entry

	// 处理子目录（CommonPrefixes）
	for _, cp := range result.CommonPrefixes {
		fullPath := strings.TrimPrefix(*cp.Prefix, s.prefix)
		fullPath = strings.TrimSuffix(fullPath, "/")
		name := fullPath
		if idx := strings.LastIndex(fullPath, "/"); idx >= 0 {
			name = fullPath[idx+1:]
		}
		entries = append(entries, Entry{
			Name:  name,
			Path:  fullPath,
			IsDir: true,
		})
	}

	// 处理文件
	for _, obj := range result.Contents {
		key := strings.TrimPrefix(*obj.Key, s.prefix)
		// 跳过目录占位符（以 / 结尾的空对象）
		if key == "" || strings.HasSuffix(key, "/") {
			continue
		}
		name := key
		if idx := strings.LastIndex(key, "/"); idx >= 0 {
			name = key[idx+1:]
		}
		entries = append(entries, Entry{
			Name:    name,
			Path:    key,
			IsDir:   false,
			ModTime: *obj.LastModified,
		})
	}

	return entries, nil
}

// Read 读取文件内容
func (s *OSSStorage) Read(path string) (string, time.Time, error) {
	key := s.fullKey(path)

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(context.TODO(), input)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("读取对象失败: %w", err)
	}
	defer result.Body.Close()

	body, err := io.ReadAll(result.Body)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("读取对象内容失败: %w", err)
	}

	modTime := time.Now()
	if result.LastModified != nil {
		modTime = *result.LastModified
	}

	return string(body), modTime, nil
}

// Write 写入文件内容
func (s *OSSStorage) Write(path string, content string) error {
	key := s.fullKey(path)

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte(content)),
	}

	_, err := s.client.PutObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("写入对象失败: %w", err)
	}

	return nil
}

// Mkdir 创建目录
// 对象存储没有真正的目录概念，创建一个以 / 结尾的空对象作为占位符
func (s *OSSStorage) Mkdir(path string) error {
	key := s.fullKey(path)
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte{}),
	}

	_, err := s.client.PutObject(context.TODO(), input)
	return err
}

// Delete 删除文件或目录
func (s *OSSStorage) Delete(path string) error {
	key := s.fullKey(path)

	// 先检查是否为"目录"（以 / 结尾的 key 或有子对象）
	dirKey := key
	if !strings.HasSuffix(dirKey, "/") {
		dirKey += "/"
	}

	// 列出该前缀下的所有对象
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(dirKey),
	}

	var objectIds []types.ObjectIdentifier
	paginator := s3.NewListObjectsV2Paginator(s.client, listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("列出对象失败: %w", err)
		}
		for _, obj := range page.Contents {
			objectIds = append(objectIds, types.ObjectIdentifier{Key: obj.Key})
		}
	}

	// 如果有子对象，批量删除
	if len(objectIds) > 0 {
		// 分批删除（每批最多 1000 个）
		for i := 0; i < len(objectIds); i += 1000 {
			end := i + 1000
			if end > len(objectIds) {
				end = len(objectIds)
			}
			batch := objectIds[i:end]

			deleteInput := &s3.DeleteObjectsInput{
				Bucket: aws.String(s.bucket),
				Delete: &types.Delete{
					Objects: batch,
					Quiet:   aws.Bool(true),
				},
			}

			_, err := s.client.DeleteObjects(context.TODO(), deleteInput)
			if err != nil {
				return fmt.Errorf("批量删除对象失败: %w", err)
			}
		}
	}

	// 同时删除文件本身（可能是文件而非目录）
	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("删除对象失败: %w", err)
	}

	// 删除目录占位符
	_, _ = s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(dirKey),
	})

	return nil
}

// GetModTime 获取文件修改时间
func (s *OSSStorage) GetModTime(path string) (time.Time, error) {
	key := s.fullKey(path)

	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.HeadObject(context.TODO(), input)
	if err != nil {
		return time.Time{}, fmt.Errorf("获取对象信息失败: %w", err)
	}

	if result.LastModified != nil {
		return *result.LastModified, nil
	}
	return time.Now(), nil
}

// Exists 检查路径是否存在
func (s *OSSStorage) Exists(path string) (bool, error) {
	key := s.fullKey(path)

	_, err := s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}

	// 检查是否为目录
	dirKey := key
	if !strings.HasSuffix(dirKey, "/") {
		dirKey += "/"
	}

	listInput := &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		Prefix:  aws.String(dirKey),
		MaxKeys: aws.Int32(1),
	}

	result, err := s.client.ListObjectsV2(context.TODO(), listInput)
	if err != nil {
		return false, err
	}

	return result.KeyCount != nil && *result.KeyCount > 0, nil
}

// Rename 重命名文件或目录（复制+删除）
func (s *OSSStorage) Rename(oldPath, newPath string) error {
	oldKey := s.fullKey(oldPath)
	newKey := s.fullKey(newPath)

	// 检查是否是目录
	isDir := false
	dirOldKey := oldKey
	if !strings.HasSuffix(dirOldKey, "/") {
		dirOldKey += "/"
	}

	listInput := &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		Prefix:  aws.String(dirOldKey),
		MaxKeys: aws.Int32(1),
	}
	result, err := s.client.ListObjectsV2(context.TODO(), listInput)
	if err == nil && result.KeyCount != nil && *result.KeyCount > 0 {
		isDir = true
	}

	if isDir {
		// 目录：需要复制所有子对象
		return s.renameDirectory(oldKey, newKey)
	}

	// 文件：直接复制
	_, err = s.client.CopyObject(context.TODO(), &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(newKey),
		CopySource: aws.String(s.bucket + "/" + oldKey),
	})
	if err != nil {
		return fmt.Errorf("复制对象失败: %w", err)
	}

	// 删除原对象
	_, err = s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(oldKey),
	})
	return err
}

// renameDirectory 重命名目录
func (s *OSSStorage) renameDirectory(oldKey, newKey string) error {
	// 确保以 / 结尾
	if !strings.HasSuffix(oldKey, "/") {
		oldKey += "/"
	}
	if !strings.HasSuffix(newKey, "/") {
		newKey += "/"
	}

	// 列出所有子对象
	var objectKeys []string
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(oldKey),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("列出对象失败: %w", err)
		}
		for _, obj := range page.Contents {
			objectKeys = append(objectKeys, *obj.Key)
		}
	}

	// 复制所有对象
	for _, oldObjKey := range objectKeys {
		// 计算新的 key
		relativePath := strings.TrimPrefix(oldObjKey, oldKey)
		newObjKey := newKey + relativePath

		_, err := s.client.CopyObject(context.TODO(), &s3.CopyObjectInput{
			Bucket:     aws.String(s.bucket),
			Key:        aws.String(newObjKey),
			CopySource: aws.String(s.bucket + "/" + oldObjKey),
		})
		if err != nil {
			return fmt.Errorf("复制对象失败: %w", err)
		}
	}

	// 删除所有旧对象
	var objectIds []types.ObjectIdentifier
	for _, key := range objectKeys {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: aws.String(key)})
	}

	// 分批删除
	for i := 0; i < len(objectIds); i += 1000 {
		end := i + 1000
		if end > len(objectIds) {
			end = len(objectIds)
		}
		batch := objectIds[i:end]

		_, err := s.client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{
				Objects: batch,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return fmt.Errorf("删除对象失败: %w", err)
		}
	}

	return nil
}

// Type 返回存储类型标识
func (s *OSSStorage) Type() string {
	return "oss"
}
