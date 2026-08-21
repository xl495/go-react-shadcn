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
	a.saveUserAvatar(c, claimsKind(claims), claims.UserID)
}

func (a *App) handleUploadUserAvatar(c *gin.Context) {
	user, found := a.loadUserInScope(c, c.Param("id"))
	if !found {
		return
	}
	a.saveUserAvatar(c, user.Kind, user.ID)
}

func (a *App) saveUserAvatar(c *gin.Context, kind string, userID uint) {
	var existing models.User
	if err := a.accounts(kind).Select("avatar").First(&existing, userID).Error; err != nil {
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
	defer func() { _ = src.Close() }()
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
	if err := os.MkdirAll(dir, 0o750); err != nil {
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	filename := fmt.Sprintf("%d_%s%s", userID, uuid.NewString(), ext)
	dstPath := filepath.Join(dir, filename)
	if filepath.Dir(dstPath) != dir {
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	// filename is generated (user id + uuid + detected ext), not caller-supplied.
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec
	if err != nil {
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	if err := dst.Close(); err != nil {
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}

	url := "/uploads/avatars/" + filename
	if err := a.accounts(kind).Where("id = ?", userID).Update("avatar", url).Error; err != nil {
		_ = os.Remove(dstPath)
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	if oldAvatar != "" && oldAvatar != url {
		removeUploadedFile(a.Cfg.UploadDir, oldAvatar)
	}
	var user models.User
	if err := a.loadAccount(kind, &user, userID); err != nil {
		fail(c, http.StatusInternalServerError, 50050, "failed to store avatar")
		return
	}
	ok(c, a.toUserDTO(user))
}

func validAvatarURL(url string) bool {
	if url == "" || strings.Contains(url, "..") || strings.Contains(url, "\\") {
		return false
	}
	const prefix = "/uploads/avatars/"
	if !strings.HasPrefix(url, prefix) {
		return false
	}
	name := strings.TrimPrefix(url, prefix)
	return name != "" && !strings.Contains(name, "/")
}

func removeUploadedFile(uploadDir, url string) {
	if !validAvatarURL(url) {
		return
	}
	root, err := filepath.Abs(uploadDir)
	if err != nil {
		return
	}
	rel := strings.TrimPrefix(url, "/uploads/")
	path, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return
	}
	sep := string(filepath.Separator)
	if path != root && !strings.HasPrefix(path, root+sep) {
		return
	}
	_ = os.Remove(path)
}
