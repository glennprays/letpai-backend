package request

// CreateContactGroupRequest represents the request to create a contact group
type CreateContactGroupRequest struct {
	Name      string `json:"name" validate:"required,min=1,max=50"`
	Color     string `json:"color" validate:"required,min=1,max=20"`
	SortOrder int    `json:"sort_order"`
}

// UpdateContactGroupRequest represents the request to update a contact group
type UpdateContactGroupRequest struct {
	Name      *string `json:"name" validate:"omitempty,min=1,max=50"`
	Color     *string `json:"color" validate:"omitempty,min=1,max=20"`
	SortOrder *int    `json:"sort_order"`
}
