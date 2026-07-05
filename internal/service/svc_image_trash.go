package service

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/haierkeys/custom-image-gateway/global"
	"github.com/haierkeys/custom-image-gateway/pkg/code"
	"github.com/haierkeys/custom-image-gateway/pkg/fileurl"
	"github.com/haierkeys/custom-image-gateway/pkg/storage"
)

type TrashImageParams struct {
	ID       int64  `json:"id" form:"id"`
	ImageURL string `json:"imageUrl" form:"imageUrl" binding:"required"`
}

type TrashImageResult struct {
	ImageURL   string `json:"imageUrl"`
	SourcePath string `json:"sourcePath"`
	TrashPath  string `json:"trashPath"`
}

type localTrashConfig struct {
	AccessURLPrefix string
	SavePath        string
	CustomPath      string
}

func (svc *Service) TrashImage(params *TrashImageParams) (*TrashImageResult, error) {
	if !global.Config.LocalFS.IsEnabled {
		return nil, code.ErrorLocalFSDisabled
	}

	return trashLocalImage(params, &localTrashConfig{
		AccessURLPrefix: global.Config.App.UploadUrlPre,
		SavePath:        global.Config.LocalFS.SavePath,
	})
}

func (svc *Service) TrashUserImage(uid int64, params *TrashImageParams) (*TrashImageResult, error) {
	if !global.Config.User.IsEnabled {
		return nil, code.ErrorMultiUserPublicAPIClosed
	}

	daoCloudConfig, err := svc.getUserUploadConfig(uid, &ClientUploadParams{ID: params.ID})
	if err != nil {
		return nil, err
	}

	if daoCloudConfig.Type != storage.LOCAL {
		return nil, code.ErrorInvalidStorageType
	}

	if err := storage.IsUserEnabled(daoCloudConfig.Type); err != nil {
		return nil, err
	}

	return trashLocalImage(params, &localTrashConfig{
		AccessURLPrefix: daoCloudConfig.AccessURLPrefix,
		SavePath:        getUserLocalSavePath(daoCloudConfig),
		CustomPath:      daoCloudConfig.CustomPath,
	})
}

func (svc *Service) RestoreImage(params *TrashImageParams) (*TrashImageResult, error) {
	if !global.Config.LocalFS.IsEnabled {
		return nil, code.ErrorLocalFSDisabled
	}

	return restoreLocalImage(params, &localTrashConfig{
		AccessURLPrefix: global.Config.App.UploadUrlPre,
		SavePath:        global.Config.LocalFS.SavePath,
	})
}

func (svc *Service) RestoreUserImage(uid int64, params *TrashImageParams) (*TrashImageResult, error) {
	if !global.Config.User.IsEnabled {
		return nil, code.ErrorMultiUserPublicAPIClosed
	}

	daoCloudConfig, err := svc.getUserUploadConfig(uid, &ClientUploadParams{ID: params.ID})
	if err != nil {
		return nil, err
	}

	if daoCloudConfig.Type != storage.LOCAL {
		return nil, code.ErrorInvalidStorageType
	}

	if err := storage.IsUserEnabled(daoCloudConfig.Type); err != nil {
		return nil, err
	}

	return restoreLocalImage(params, &localTrashConfig{
		AccessURLPrefix: daoCloudConfig.AccessURLPrefix,
		SavePath:        getUserLocalSavePath(daoCloudConfig),
		CustomPath:      daoCloudConfig.CustomPath,
	})
}

func trashLocalImage(params *TrashImageParams, cfg *localTrashConfig) (*TrashImageResult, error) {
	if params == nil || strings.TrimSpace(params.ImageURL) == "" {
		return nil, code.ErrorInvalidImageURL
	}

	sourcePath, trashPath, err := resolveLocalTrashPaths(strings.TrimSpace(params.ImageURL), cfg)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(trashPath), os.ModePerm); err != nil {
		return nil, err
	}

	targetPath, err := ensureUniqueTrashPath(trashPath)
	if err != nil {
		return nil, err
	}

	if err := os.Rename(sourcePath, targetPath); err != nil {
		return nil, err
	}

	return &TrashImageResult{
		ImageURL:   params.ImageURL,
		SourcePath: sourcePath,
		TrashPath:  targetPath,
	}, nil
}

func restoreLocalImage(params *TrashImageParams, cfg *localTrashConfig) (*TrashImageResult, error) {
	if params == nil || strings.TrimSpace(params.ImageURL) == "" {
		return nil, code.ErrorInvalidImageURL
	}

	sourcePath, trashPath, err := resolveLocalTrashPathsWithoutSourceCheck(strings.TrimSpace(params.ImageURL), cfg)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(sourcePath); err == nil {
		return nil, code.ErrorImageRestoreConflict
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	actualTrashPath, err := findExistingTrashPath(trashPath)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(sourcePath), os.ModePerm); err != nil {
		return nil, err
	}

	if err := os.Rename(actualTrashPath, sourcePath); err != nil {
		return nil, err
	}

	return &TrashImageResult{
		ImageURL:   params.ImageURL,
		SourcePath: sourcePath,
		TrashPath:  actualTrashPath,
	}, nil
}

func resolveLocalTrashPaths(imageURL string, cfg *localTrashConfig) (string, string, error) {
	return resolveLocalTrashPathsInternal(imageURL, cfg, true)
}

func resolveLocalTrashPathsWithoutSourceCheck(imageURL string, cfg *localTrashConfig) (string, string, error) {
	return resolveLocalTrashPathsInternal(imageURL, cfg, false)
}

func resolveLocalTrashPathsInternal(imageURL string, cfg *localTrashConfig, requireSourceExists bool) (string, string, error) {
	if cfg == nil || strings.TrimSpace(cfg.AccessURLPrefix) == "" {
		return "", "", code.ErrorInvalidAccessURLPrefix
	}

	prefixURL, err := url.Parse(cfg.AccessURLPrefix)
	if err != nil || prefixURL.Host == "" {
		return "", "", code.ErrorInvalidAccessURLPrefix
	}

	parsedURL, err := url.Parse(imageURL)
	if err != nil || parsedURL.Host == "" {
		return "", "", code.ErrorInvalidImageURL
	}

	if !strings.EqualFold(parsedURL.Host, prefixURL.Host) || !strings.EqualFold(parsedURL.Scheme, prefixURL.Scheme) {
		return "", "", code.ErrorImageURLPrefixMismatch
	}

	incomingPath, err := url.PathUnescape(strings.TrimPrefix(parsedURL.EscapedPath(), "/"))
	if err != nil {
		return "", "", code.ErrorInvalidImageURL
	}

	prefixPath, err := url.PathUnescape(strings.TrimPrefix(prefixURL.EscapedPath(), "/"))
	if err != nil {
		return "", "", code.ErrorInvalidAccessURLPrefix
	}

	if prefixPath != "" {
		if incomingPath == prefixPath {
			incomingPath = ""
		} else if strings.HasPrefix(incomingPath, prefixPath+"/") {
			incomingPath = strings.TrimPrefix(incomingPath, prefixPath+"/")
		} else {
			return "", "", code.ErrorImageURLPrefixMismatch
		}
	}

	basePath := strings.TrimSpace(cfg.SavePath)
	relativeKey := filepath.ToSlash(strings.TrimLeft(incomingPath, "/"))
	customPath := strings.TrimSpace(cfg.CustomPath)

	if customPath != "" {
		if fileurl.IsAbsPath(customPath) {
			basePath = customPath
		} else {
			basePath = customPath
			customPathURL := filepath.ToSlash(strings.Trim(customPath, "/"))
			if relativeKey == customPathURL {
				return "", "", code.ErrorImagePathInvalid
			}
			if customPathURL != "" && strings.HasPrefix(relativeKey, customPathURL+"/") {
				relativeKey = strings.TrimPrefix(relativeKey, customPathURL+"/")
			} else {
				return "", "", code.ErrorImageURLPrefixMismatch
			}
		}
	} else {
		savePathURL := filepath.ToSlash(strings.Trim(basePath, "/"))
		if savePathURL != "" && !fileurl.IsAbsPath(basePath) {
			if strings.HasPrefix(relativeKey, savePathURL+"/") {
				relativeKey = strings.TrimPrefix(relativeKey, savePathURL+"/")
			} else {
				return "", "", code.ErrorImageURLPrefixMismatch
			}
		}
	}

	relativeKey = filepath.ToSlash(strings.TrimLeft(relativeKey, "/"))
	if relativeKey == "" || fileurl.IsAbsPath(relativeKey) {
		return "", "", code.ErrorImagePathInvalid
	}

	baseAbs, err := filepath.Abs(basePath)
	if err != nil {
		return "", "", err
	}
	baseAbs = filepath.Clean(baseAbs)

	sourcePath := filepath.Join(baseAbs, filepath.FromSlash(relativeKey))
	if !isWithinBasePath(baseAbs, sourcePath) {
		return "", "", code.ErrorImagePathInvalid
	}

	if requireSourceExists {
		info, err := os.Stat(sourcePath)
		if err != nil {
			if os.IsNotExist(err) {
				return "", "", code.ErrorImageFileNotFound
			}
			return "", "", err
		}
		if info.IsDir() {
			return "", "", code.ErrorImagePathInvalid
		}
	}

	trashBase := filepath.Join(baseAbs, ".trash")
	trashPath := filepath.Join(trashBase, filepath.FromSlash(relativeKey))
	if !isWithinBasePath(trashBase, trashPath) {
		return "", "", code.ErrorImagePathInvalid
	}

	return sourcePath, trashPath, nil
}

func ensureUniqueTrashPath(targetPath string) (string, error) {
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return targetPath, nil
	} else if err != nil {
		return "", err
	}

	ext := filepath.Ext(targetPath)
	base := strings.TrimSuffix(targetPath, ext)
	return fmt.Sprintf("%s-%d%s", base, time.Now().Unix(), ext), nil
}

func findExistingTrashPath(targetPath string) (string, error) {
	if _, err := os.Stat(targetPath); err == nil {
		return targetPath, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	dir := filepath.Dir(targetPath)
	baseName := filepath.Base(targetPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", code.ErrorImageTrashFileNotFound
		}
		return "", err
	}

	prefix := nameWithoutExt + "-"
	var latestPath string
	var latestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		entryName := entry.Name()
		if !strings.HasPrefix(entryName, prefix) || filepath.Ext(entryName) != ext {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		if latestPath == "" || info.ModTime().After(latestTime) {
			latestPath = filepath.Join(dir, entryName)
			latestTime = info.ModTime()
		}
	}

	if latestPath == "" {
		return "", code.ErrorImageTrashFileNotFound
	}

	return latestPath, nil
}

func isWithinBasePath(basePath string, targetPath string) bool {
	baseClean := filepath.Clean(basePath)
	targetClean := filepath.Clean(targetPath)
	rel, err := filepath.Rel(baseClean, targetClean)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ""
}
