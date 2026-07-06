package service

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"mime/multipart"
	"path"
	"strings"
	"time"

	"github.com/haierkeys/custom-image-gateway/global"
	"github.com/haierkeys/custom-image-gateway/internal/dao"
	"github.com/haierkeys/custom-image-gateway/pkg/code"
	"github.com/haierkeys/custom-image-gateway/pkg/convert"
	"github.com/haierkeys/custom-image-gateway/pkg/fileurl"
	"github.com/haierkeys/custom-image-gateway/pkg/storage"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/avif"
	"github.com/pkg/errors"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"gorm.io/gorm"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type FileInfo struct {
	ImageTitle string   `json:"imageTitle"`
	ImageUrl   string   `json:"imageUrl"`
	UseStore   []string `json:"useStore"`
}

// 上传文件
type ClientUploadParams struct {
	ID             int64  `form:"id"`
	Key            string `form:"key"`
	Type           string `form:"type"`
	Width          int    `form:"width"`
	Height         int    `form:"height"`
	RenameOnUpload bool   `form:"renameOnUpload"`
	RenamePattern  string `form:"renamePattern"`
	SourceMtime    int64  `form:"sourceMtime"`
}

// UploadFile 上传文件
func (svc *Service) UploadFile(file multipart.File, fileHeader *multipart.FileHeader, params *ClientUploadParams) (*FileInfo, error) {

	// 上传文件名
	var fileName = fileurl.GetFileNameOrRandom(fileHeader.Filename)

	// 检查文件后缀
	if !fileurl.IsContainExt(fileurl.ImageType, fileName, global.Config.App.UploadAllowExts) {
		return nil, errors.New("file suffix is not supported.")
	}
	// 检查文件大小
	if fileurl.IsFileSizeAllowed(fileurl.ImageType, file, global.Config.App.UploadMaxSize) {
		return nil, errors.New("exceeded maximum file limit.")
	}

	var fileKey = fileurl.GetDatePath(global.Config.App.UploadDatePath) + fileName
	var fileType = fileHeader.Header.Get("Content-Type")
	var dstFileKey string

	// 压缩
	writer, fileKey, fileType, err := imageResize(params, file, fileKey, fileType)
	if err != nil {
		return nil, err
	}

	useStore := []string{}
	for sType := range storage.StorageTypeMap {
		config := map[string]any{}
		if sType == storage.LOCAL {
			_ = convert.StructToMap(global.Config.LocalFS, config)
		} else if sType == storage.OSS {
			_ = convert.StructToMap(global.Config.OSS, config)
		} else if sType == storage.R2 {
			_ = convert.StructToMap(global.Config.CloudflueR2, config)
		} else if sType == storage.S3 {
			_ = convert.StructToMap(global.Config.AWSS3, config)
		} else if sType == storage.MinIO {
			_ = convert.StructToMap(global.Config.MinIO, config)
		} else if sType == storage.WebDAV {
			_ = convert.StructToMap(global.Config.WebDAV, config)
		} else {
			continue
		}
		if !config["IsEnabled"].(bool) {
			continue
		}
		ins, err := storage.NewClient(sType, config)
		if err != nil {
			return nil, err
		}

		targetFileKey := fileKey
		if params != nil && params.RenameOnUpload {
			targetFileKey, err = buildUploadFileKey(ins, fileKey, params)
			if err != nil {
				return nil, err
			}
		}

		reader := bytes.NewReader(writer.Bytes())
		dstFileKey, err = ins.SendFile(targetFileKey, reader, fileType)
		if err != nil {
			return nil, err
		}
		useStore = append(useStore, sType)
	}
	accessUrl := fileurl.PathSuffixCheckAdd(global.Config.App.UploadUrlPre, "/") + fileurl.UrlEscape(dstFileKey)

	return &FileInfo{ImageTitle: fileHeader.Filename, ImageUrl: accessUrl, UseStore: useStore}, nil
}

func (svc *Service) UserUploadFile(uid int64, file multipart.File, fileHeader *multipart.FileHeader, params *ClientUploadParams) (*FileInfo, error) {

	if !global.Config.User.IsEnabled {
		return nil, code.ErrorMultiUserPublicAPIClosed
	}

	// 上传文件名
	var fileName = fileurl.GetFileNameOrRandom(fileHeader.Filename)

	// 检查文件后缀
	if !fileurl.IsContainExt(fileurl.ImageType, fileName, global.Config.App.UploadAllowExts) {
		return nil, errors.New("file suffix is not supported.")
	}
	// 检查文件大小
	if fileurl.IsFileSizeAllowed(fileurl.ImageType, file, global.Config.App.UploadMaxSize) {
		return nil, errors.New("exceeded maximum file limit.")
	}

	var fileKey = fileurl.GetDatePath(global.Config.App.UploadDatePath) + fileName
	var fileType = fileHeader.Header.Get("Content-Type")
	var dstFileKey string

	// 压缩
	writer, fileKey, fileType, err := imageResize(params, file, fileKey, fileType)
	if err != nil {
		return nil, err
	}

	daoCloudConfig, err := svc.getUserUploadConfig(uid, params)
	if err != nil {
		return nil, err
	}

	var userCloudConfig = convert.StructToMapByReflect(daoCloudConfig)

	// 检查云存储类型是否启用
	if err := storage.IsUserEnabled(daoCloudConfig.Type); err != nil {
		return nil, err
	}

	userCloudConfig["SavePath"] = getUserLocalSavePath(daoCloudConfig)

	ins, err := storage.NewClient(daoCloudConfig.Type, userCloudConfig)
	if err != nil {
		return nil, err
	}

	targetFileKey := fileKey
	if params != nil && params.RenameOnUpload {
		targetFileKey, err = buildUploadFileKey(ins, fileKey, params)
		if err != nil {
			return nil, err
		}
	}

	reader := bytes.NewReader(writer.Bytes())
	dstFileKey, err = ins.SendFile(targetFileKey, reader, fileType)
	if err != nil {
		return nil, err
	}

	useStore := []string{daoCloudConfig.Type}

	// 	accessUrl := fileurl.PathSuffixCheckAdd(userCloudConfig["AccessURLPrefix"].(string), "/") + fileurl.UrlEscape(dstFileKey)

	// 	return &FileInfo{ImageTitle: fileHeader.Filename, ImageUrl: accessUrl, UseStore: useStore}, nil
	// }

	accessUrl := buildUserAccessURL(daoCloudConfig, dstFileKey, targetFileKey)

	return &FileInfo{ImageTitle: fileHeader.Filename, ImageUrl: accessUrl, UseStore: useStore}, nil
}

func (svc *Service) getUserUploadConfig(uid int64, params *ClientUploadParams) (*dao.CloudConfig, error) {
	return resolveUserUploadConfig(svc, uid, params, svc.dao.GetById, svc.dao.GetEnableByUId)
}

func resolveUserUploadConfig(
	_ *Service,
	uid int64,
	params *ClientUploadParams,
	getByID func(id int64, uid int64) (*dao.CloudConfig, error),
	getEnabledByUID func(uid int64) (*dao.CloudConfig, error),
) (*dao.CloudConfig, error) {
	if params != nil && params.ID > 0 {
		cfg, err := getByID(params.ID, uid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, code.ErrorUserCloudConfigIDNotFound
			}
			return nil, err
		}
		return cfg, nil
	}

	cfg, err := getEnabledByUID(uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, code.ErrorUserCloudConfigNotFound
		}
		return nil, err
	}
	return cfg, nil
}

func getUserLocalSavePath(cfg *dao.CloudConfig) string {
	if cfg != nil && cfg.Type == storage.LOCAL && strings.TrimSpace(cfg.CustomPath) != "" {
		return cfg.CustomPath
	}
	return global.Config.LocalFS.SavePath
}

func buildUserAccessURL(cfg *dao.CloudConfig, dstFileKey string, fileKey string) string {
	if cfg == nil {
		return fileurl.PathSuffixCheckAdd(global.Config.App.UploadUrlPre, "/") + fileurl.UrlEscape(dstFileKey)
	}

	accessPrefix := fileurl.PathSuffixCheckAdd(cfg.AccessURLPrefix, "/")
	if cfg.Type != storage.LOCAL {
		return accessPrefix + fileurl.UrlEscape(dstFileKey)
	}

	localURLPath := buildUserLocalURLPath(cfg.CustomPath, fileKey)
	return accessPrefix + fileurl.UrlEscape(localURLPath)
}

func buildUserLocalURLPath(customPath string, fileKey string) string {
	fileKey = normalizeURLPath(fileKey)
	customPath = strings.TrimSpace(customPath)
	if customPath == "" {
		return normalizeURLPath(fileurl.PathSuffixCheckAdd(global.Config.LocalFS.SavePath, "/") + fileKey)
	}
	if fileurl.IsAbsPath(customPath) {
		return fileKey
	}
	return normalizeURLPath(fileurl.PathSuffixCheckAdd(customPath, "/") + fileKey)
}

func normalizeURLPath(v string) string {
	return strings.ReplaceAll(v, "\\", "/")
}

func buildUploadFileKey(ins storage.Storager, originalFileKey string, params *ClientUploadParams) (string, error) {
	if params == nil || !params.RenameOnUpload {
		return originalFileKey, nil
	}

	if params.RenamePattern != "" && params.RenamePattern != "second-only-append-random-on-conflict" {
		return originalFileKey, nil
	}

	baseDir := path.Dir(normalizeURLPath(originalFileKey))
	if baseDir == "." {
		baseDir = ""
	}

	ext := strings.ToLower(fileurl.GetFileExt(originalFileKey))
	timestamp := resolveUploadTimestamp(params.SourceMtime).Format("20060102150405")
	baseName := timestamp + ext
	baseFileKey := joinPath(baseDir, baseName)

	checker, ok := ins.(storage.ObjectExistChecker)
	if !ok {
		return baseFileKey, nil
	}

	exists, err := checker.ObjectExists(baseFileKey)
	if err != nil {
		return "", err
	}
	if !exists {
		return baseFileKey, nil
	}

	for attempt := 0; attempt < 3; attempt++ {
		candidate := joinPath(baseDir, timestamp+"-"+randomAlphaNum(6)+ext)
		exists, err = checker.ObjectExists(candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}

	return "", errors.New("failed to generate a unique upload file name")
}

func resolveUploadTimestamp(sourceMtime int64) time.Time {
	if sourceMtime <= 0 {
		return time.Now()
	}

	t := time.UnixMilli(sourceMtime)
	if t.Year() < 2000 || t.Year() > 3000 {
		return time.Now()
	}
	return t
}

func joinPath(dir string, fileName string) string {
	if strings.TrimSpace(dir) == "" {
		return fileName
	}
	return strings.TrimSuffix(dir, "/") + "/" + fileName
}

func randomAlphaNum(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		builder.WriteByte(alphabet[rand.Intn(len(alphabet))])
	}
	return builder.String()
}

// imageResize 压缩图片
// 默认裁剪 | 居中裁剪 | 固定尺寸拉伸 | 固定尺寸等比缩放不裁切 | 不处理
// type: "fill-topleft" | "fill-center" | "resize" | "fit" | "none";
func imageResize(params *ClientUploadParams, file multipart.File, fileKey string, fileType string) (*bytes.Buffer, string, string, error) {

	var writer = &bytes.Buffer{}
	// 压缩
	_, err := file.Seek(0, 0)
	if err != nil {
		return nil, fileKey, fileType, err
	}

	if params == nil || params.Type == "" || params.Type == "none" {
		_, err = io.Copy(writer, file)
		if err != nil {
			return nil, fileKey, fileType, err
		}
		return writer, fileKey, fileType, nil
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, fileKey, fileType, err
	}

	img, fileRealType, err := image.Decode(file)

	if err != nil {
		return nil, fileKey, fileType, err
	}

	var imgSize = img.Bounds().Size()

	// 服务器强制限制图片的宽度和高度
	var imageMaxWidth = global.Config.App.ImageMaxSizeWidth
	var imageMaxHeight = global.Config.App.ImageMaxSizeHeight
	var newWidth, newHeight int
	var newImage image.Image
	var isNewImage bool

	if params.Type == "fill-topleft" {
		if params.Width < imageMaxWidth || imageMaxWidth == 0 {
			newWidth = params.Width
		} else {
			newWidth = imageMaxWidth
		}
		if params.Height < imageMaxHeight || imageMaxHeight == 0 {
			newHeight = params.Height
		} else {
			newHeight = imageMaxHeight
		}
		newImage = imaging.Fill(img, newWidth, newHeight, imaging.TopLeft, imaging.Lanczos)
		isNewImage = true
	} else if params.Type == "fill-center" {
		if params.Width < imageMaxWidth || imageMaxWidth == 0 {
			newWidth = params.Width
		} else {
			newWidth = imageMaxWidth
		}
		if params.Height < imageMaxHeight || imageMaxHeight == 0 {
			newHeight = params.Height
		} else {
			newHeight = imageMaxHeight
		}
		// newImage = imaging.Fit(img, newWidth, newHeight, imaging.Lanczos)
		newImage = imaging.Fill(img, newWidth, newHeight, imaging.Center, imaging.Lanczos)
		isNewImage = true
	} else if params.Type == "resize" {
		if params.Width < imageMaxWidth || imageMaxWidth == 0 {
			newWidth = params.Width
		} else {
			newWidth = imageMaxWidth
		}
		if params.Height < imageMaxHeight || imageMaxHeight == 0 {
			newHeight = params.Height
		} else {
			newHeight = imageMaxHeight
		}
		if params.Width != 0 && params.Height != 0 && (imgSize.X != newWidth || imgSize.Y != newHeight) {
			newImage = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
			isNewImage = true
		}
	} else if params.Type == "fit" {
		if params.Width < imageMaxWidth || imageMaxWidth == 0 {
			newWidth = params.Width
		} else {
			newWidth = imageMaxWidth
		}
		if params.Height < imageMaxHeight || imageMaxHeight == 0 {
			newHeight = params.Height
		} else {
			newHeight = imageMaxHeight
		}
		if (imgSize.X != newWidth || imgSize.Y != newHeight) && (newWidth != 0 || newHeight != 0) {
			if newWidth == 0 || newHeight == 0 {
				newImage = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
			} else {
				newImage = imaging.Fit(img, newWidth, newHeight, imaging.Lanczos)
			}
			isNewImage = true
		}
	}
	if isNewImage {
		// 调整图片大小
		switch fileRealType {
		case "png":
			err = png.Encode(writer, newImage)
		case "gif":
			err = gif.Encode(writer, newImage, &gif.Options{NumColors: 256})
		case "jpeg", "jpg":
			err = jpeg.Encode(writer, newImage, &jpeg.Options{Quality: global.Config.App.ImageQuality})
		case "bmp":
			err = bmp.Encode(writer, newImage)
		case "tif", "tiff":
			err = tiff.Encode(writer, newImage, nil)
		case "webp":
			// 暂时不支持
			fileType = "image/jpg"
			ext := fileurl.GetFileExt(fileKey)
			fileKey = fileKey[0:len(fileKey)-len(ext)] + ".jpg"
			err = jpeg.Encode(writer, newImage, &jpeg.Options{Quality: global.Config.App.ImageQuality})
		case "avif":
			err = avif.Encode(writer, newImage, avif.Options{Quality: global.Config.App.ImageQuality})
		default:
			return nil, fileKey, fileType, errors.New("Unknown image type:" + fileRealType)
		}
		if err != nil {
			return nil, fileKey, fileType, err
		}
	} else {
		_, err = file.Seek(0, 0)
		if err != nil {
			return nil, fileKey, fileType, err
		}
		_, err = io.Copy(writer, file)
		if err != nil {
			return nil, fileKey, fileType, err
		}
	}
	return writer, fileKey, fileType, nil
}
