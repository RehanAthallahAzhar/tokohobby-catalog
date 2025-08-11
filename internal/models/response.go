package models

// ErrorResponse digunakan untuk error standar (400, 404, 500, dll)
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse untuk response sukses standar (tanpa data)
type SuccessResponse struct {
	Username string      `json:"username"`
	Message  string      `json:"message"`
	Data     interface{} `json:"data"`
}

// DataResponse untuk response sukses dengan data tunggal
type DataResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ListResponse untuk response sukses berisi banyak data
type ListResponse struct {
	Message string        `json:"message"`
	Data    []interface{} `json:"data"`
	Count   int           `json:"count"`
}

// PaginatedResponse untuk response dengan pagination
type PaginatedResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Paging  PagingInfo  `json:"paging"`
}

type PagingInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}
