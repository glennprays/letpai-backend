package admin

type ListAdminsResponse struct {
	Admins  []*AdminAdminResponse
	Message string `json:"message"`
}

type AdminAdminResponse struct {
	AdminID        string `json:"admin_id"`
	WhatsAppNumber string `json:"whatsapp_number"`
	FullName       string `json:"full_name"`
	Role           string `json:"role"`
	IsActive       bool   `json:"is_active"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type AdminRequest struct {
	FullName string `json:"full_name" validate:"omitempty,max=100"`
}

type UpdateAdminResponse struct {
	Message string `json:"message"`
}

type DeleteAdminRequest struct{}

type DeleteAdminResponse struct {
	Message string `json:"message"`
}
