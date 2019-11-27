package controllers

import (
	"context"
	"github.com/HFO4/cloudreve/models"
	"github.com/HFO4/cloudreve/pkg/filesystem"
	"github.com/HFO4/cloudreve/pkg/filesystem/local"
	"github.com/HFO4/cloudreve/pkg/serializer"
	"github.com/HFO4/cloudreve/pkg/util"
	"github.com/HFO4/cloudreve/service/explorer"
	"github.com/gin-gonic/gin"
	"net/url"
	"strconv"
)

// Download 文件下载
func Download(c *gin.Context) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.FileDownloadService
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.Download(ctx, c)
		if res.Code != 0 {
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

// FileUploadStream 本地策略流式上传
func FileUploadStream(c *gin.Context) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 非本地策略时拒绝上传
	if user, ok := c.Get("user"); ok && user.(*model.User).Policy.Type != "local" {
		c.JSON(200, serializer.Err(serializer.CodePolicyNotAllowed, "当前上传策略无法使用", nil))
		return
	}

	// 取得文件大小
	fileSize, err := strconv.ParseUint(c.Request.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}

	// 解码文件名和路径
	fileName, err := url.QueryUnescape(c.Request.Header.Get("X-FileName"))
	filePath, err := url.QueryUnescape(c.Request.Header.Get("X-Path"))
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}

	fileData := local.FileStream{
		MIMEType:    c.Request.Header.Get("Content-Type"),
		File:        c.Request.Body,
		Size:        fileSize,
		Name:        fileName,
		VirtualPath: util.DotPathToStandardPath(filePath),
	}

	// 创建文件系统
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err))
		return
	}

	// 给文件系统分配钩子
	fs.Use("BeforeUpload", filesystem.HookValidateFile)
	fs.Use("BeforeUpload", filesystem.HookValidateCapacity)
	fs.Use("AfterUploadCanceled", filesystem.HookDeleteTempFile)
	fs.Use("AfterUploadCanceled", filesystem.HookGiveBackCapacity)
	fs.Use("AfterUpload", filesystem.GenericAfterUpload)
	fs.Use("AfterValidateFailed", filesystem.HookDeleteTempFile)
	fs.Use("AfterValidateFailed", filesystem.HookGiveBackCapacity)

	// 执行上传
	uploadCtx := context.WithValue(ctx, filesystem.GinCtx, c)
	err = fs.Upload(uploadCtx, fileData)
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeUploadFailed, err.Error(), err))
		return
	}

	c.JSON(200, serializer.Response{
		Code: 0,
		Msg:  "Pong",
	})
}
