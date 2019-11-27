package explorer

import (
	"context"
	"github.com/HFO4/cloudreve/pkg/filesystem"
	"github.com/HFO4/cloudreve/pkg/serializer"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// FileDownloadService 文件下载服务，path为文件完整路径
type FileDownloadService struct {
	Path string `form:"path" binding:"required,min=1,max=65535"`
}

// Download 文件下载
func (service *FileDownloadService) Download(ctx context.Context, c *gin.Context) serializer.Response {
	// 创建文件系统
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}

	// 开始处理下载
	ctx = context.WithValue(ctx, filesystem.GinCtx, c)
	rs, err := fs.GetContent(ctx, service.Path)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	// 设置文件名
	c.Header("Content-Disposition", "attachment; filename=\""+fs.Target.Name+"\"")
	// 发送文件
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, rs)

	return serializer.Response{
		Code: 0,
	}
}
