// internal/dto/me.go
package dto

type UpdateMyProfileRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
}
