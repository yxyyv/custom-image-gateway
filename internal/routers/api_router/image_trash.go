package apiRouter

import (
	"github.com/haierkeys/custom-image-gateway/global"
	"github.com/haierkeys/custom-image-gateway/internal/service"
	"github.com/haierkeys/custom-image-gateway/pkg/app"
	"github.com/haierkeys/custom-image-gateway/pkg/code"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ImageTrash struct{}

func NewImageTrash() ImageTrash {
	return ImageTrash{}
}

func (u ImageTrash) Trash(c *gin.Context) {
	params := &service.TrashImageParams{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, params)

	if !valid {
		global.Logger.Error("apiRouter.ImageTrash.BindAndValid errs: %v", zap.Error(errs))
		response.ToResponse(code.ErrorInvalidParams.WithDetails(errs.ErrorsToString()).WithData(errs.MapsToString()))
		return
	}

	svc := service.New(c)
	result, err := svc.TrashImage(params)
	if err != nil {
		global.Logger.Error("svc.TrashImage err: %v", zap.Error(err))
		response.ToResponse(code.Failed.WithDetails(err.Error()))
		return
	}

	response.ToResponse(code.SuccessDelete.WithData(result))
}

func (u ImageTrash) UserTrash(c *gin.Context) {
	params := &service.TrashImageParams{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, params)

	if !valid {
		global.Logger.Error("apiRouter.ImageTrash.UserTrash.BindAndValid errs: %v", zap.Error(errs))
		response.ToResponse(code.ErrorInvalidParams.WithDetails(errs.ErrorsToString()).WithData(errs.MapsToString()))
		return
	}

	uid := app.GetUID(c)
	if uid == 0 {
		global.Logger.Error("apiRouter.ImageTrash.UserTrash err uid=0")
		response.ToResponse(code.ErrorNotUserAuthToken)
		return
	}

	svc := service.New(c)
	result, err := svc.TrashUserImage(uid, params)
	if err != nil {
		global.Logger.Error("svc.TrashUserImage err: %v", zap.Error(err))
		response.ToResponse(code.Failed.WithDetails(err.Error()))
		return
	}

	response.ToResponse(code.SuccessDelete.WithData(result))
}

func (u ImageTrash) Restore(c *gin.Context) {
	params := &service.TrashImageParams{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, params)

	if !valid {
		global.Logger.Error("apiRouter.ImageTrash.Restore.BindAndValid errs: %v", zap.Error(errs))
		response.ToResponse(code.ErrorInvalidParams.WithDetails(errs.ErrorsToString()).WithData(errs.MapsToString()))
		return
	}

	svc := service.New(c)
	result, err := svc.RestoreImage(params)
	if err != nil {
		global.Logger.Error("svc.RestoreImage err: %v", zap.Error(err))
		response.ToResponse(code.Failed.WithDetails(err.Error()))
		return
	}

	response.ToResponse(code.SuccessUpdate.WithData(result))
}

func (u ImageTrash) UserRestore(c *gin.Context) {
	params := &service.TrashImageParams{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, params)

	if !valid {
		global.Logger.Error("apiRouter.ImageTrash.UserRestore.BindAndValid errs: %v", zap.Error(errs))
		response.ToResponse(code.ErrorInvalidParams.WithDetails(errs.ErrorsToString()).WithData(errs.MapsToString()))
		return
	}

	uid := app.GetUID(c)
	if uid == 0 {
		global.Logger.Error("apiRouter.ImageTrash.UserRestore err uid=0")
		response.ToResponse(code.ErrorNotUserAuthToken)
		return
	}

	svc := service.New(c)
	result, err := svc.RestoreUserImage(uid, params)
	if err != nil {
		global.Logger.Error("svc.RestoreUserImage err: %v", zap.Error(err))
		response.ToResponse(code.Failed.WithDetails(err.Error()))
		return
	}

	response.ToResponse(code.SuccessUpdate.WithData(result))
}
