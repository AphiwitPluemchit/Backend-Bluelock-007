package models

import "math"

// PaginationParams ใช้เก็บค่าการแบ่งหน้า, ค้นหา และเรียงลำดับ
type PaginationParams struct {
	Page   int    `json:"page" query:"page"  example:"1"`      // หมายเลขหน้าที่ต้องการ
	Limit  int    `json:"limit" query:"limit" example:"10"`    // จำนวนรายการต่อหน้า
	Search string `json:"search" query:"search" example:""`    // คำค้นหา (Optional)
	SortBy string `json:"sortBy" query:"sortBy" example:"_id"` // ฟิลด์ที่ใช้เรียงลำดับ
	Order  string `json:"order" query:"order" example:"desc"`  // ทิศทางการเรียง (asc/desc)
}

// PaginatedResponse โครงสร้างการตอบกลับแบบแบ่งหน้า
type PaginatedResponse struct {
	Data        interface{} `json:"data"`
	Total       int64       `json:"total"`
	Page        int         `json:"page"`
	Limit       int         `json:"limit"`
	TotalPages  int         `json:"totalPages"`
	HasNext     bool        `json:"hasNext"`
	HasPrevious bool        `json:"hasPrevious"`
}

// DefaultPagination ค่าตั้งต้นสำหรับ Pagination
func DefaultPagination() PaginationParams {
	return PaginationParams{
		Page:   1,
		Limit:  10,
		Search: "",
		SortBy: "_id",
		Order:  "asc",
	}
}

// NewPaginatedResponse สร้าง PaginatedResponse ใหม่
func NewPaginatedResponse(data interface{}, total int64, params PaginationParams) *PaginatedResponse {
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return &PaginatedResponse{
		Data:        data,
		Total:       total,
		Page:        params.Page,
		Limit:       params.Limit,
		TotalPages:  totalPages,
		HasNext:     params.Page < totalPages,
		HasPrevious: params.Page > 1,
	}
}

// GetSkip คำนวณจำนวนรายการที่ต้องข้าม
func (p *PaginationParams) GetSkip() int64 {
	return int64((p.Page - 1) * p.Limit)
}

// GetSortOrder สร้างตัวแปรสำหรับการเรียงลำดับ
func (p *PaginationParams) GetSortOrder() map[string]int {
	order := 1 // 1 = asc, -1 = desc
	if p.Order == "desc" {
		order = -1
	}
	return map[string]int{p.SortBy: order}
}
