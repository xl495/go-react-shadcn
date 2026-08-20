package httpserver

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-react-shadcn/internal/models"
)

var avatarTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

func (a *App) handleUploadOwnAvatar(c *gin.Context) {
	claims := currentUser(c)
	a.saveUserAvatar(c, claims.UserID)
}

func (a *App) handleUploadUserAvatar(c *gin.Context) {
	var user models.User
	if err := a.DB.First(&user, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40410, "user not found")
		return
	}
	a.saveUserAvatar(c, user.ID)
}

func (a *App) saveUserAvatar(c *gin.Context, userID uint) {
	var existing models.User
	if err := a.DB.Select("avatar").First(&existing, userID).Error; err != nil {
		fail(c, http.StatusNotFound, 40410, "user not found")
		return
	}
	oldAvatar := existing.Avatar
	file, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, 40050, "avatar file required")
		return
	}
	if file.Size <= 0 || file.Size > 2<<20 {
		fail(c, http.StatusBadRequest, 40051, "avatar must be 1B-2MB")
		return
	}
	src, err := file.Open()
	if err != nil {
		fail(c, http.StatusBadRequest, 40050, "avatar file required")
		return
	}
	defer src.Close()
	head := make([]byte, 512)
	n, _ := src.Read(head)
	ctype := http.DetectContentType(head[:n])
	ext, known := avatarTypes[ctype]
	if !known {
		fail(c, http.StatusBadRequest, 40052, "unsupported image type")
		return
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		fail(c, http.StatusInternalServerError, 50050, "failed to read avatar")
		return
	}
	dir := filepath.Join(a.Cfg.UploadDir, "avatars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	filename := fmt.Sprintf("%d_%s%s", userID, uuid.NewString(), ext)
	dstPath := filepath.Join(dir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	dst.Close()

	url := "/uploads/avatars/" + filename
	if err := a.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar", url).Error; err != nil {
		_ = os.Remove(dstPath)
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	if oldAvatar != "" && oldAvatar != url {
		removeUploadedFile(a.Cfg.UploadDir, oldAvatar)
	}
	var user models.User
	if err := a.DB.Preload("Roles.Permissions").First(&user, userID).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	ok(c, toUserDTO(user))
}

func removeUploadedFile(uploadDir, url string) {
	if url == "" || !strings.HasPrefix(url, "/uploads/") {
		return
	}
	rel := strings.TrimPrefix(url, "/uploads/")
	path := filepath.Join(uploadDir, filepath.FromSlash(rel))
	_ = os.Remove(path)
}
