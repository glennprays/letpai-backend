package admin

type GetAdminsRequest struct {
	Page  int `json:"page" validate:"min=1"`
	Limit int `json:"limit" validate:"min=1"`
}

type CreateAdminRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,len=13,max=20"`
	FullName       string `json:"full_name" validate:"required,max=100"`
	Role           string `json:"role" validate:"required,oneof=super_admin,admin"`
}

type UpdateAdminRequest struct {
	FullName string `json:"full_name" validate:"omitempty,max=100"`
	Role     string `json:"role" validate:"omitempty,oneof=super_admin,admin"`
	IsActive bool   `json:"is_active" validate:"omitempty"`
}

type DeleteAdminRequest struct{}
