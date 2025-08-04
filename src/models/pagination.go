package models

// PaginationParams ใช้เก็บค่าการแบ่งหน้า, ค้นหา และเรียงลำดับ
type PaginationParams struct {
	Page   int    `json:"page" query:"page"  example:"1"`      // หมายเลขหน้าที่ต้องการ
	Limit  int    `json:"limit" query:"limit" example:"10"`    // จำนวนรายการต่อหน้า
	Search string `json:"search" query:"search" example:""`    // คำค้นหา (Optional)
	SortBy string `json:"sortBy" query:"sortBy" example:"_id"` // ฟิลด์ที่ใช้เรียงลำดับ
	Order  string `json:"order" query:"order" example:"desc"`  // ทิศทางการเรียง (asc/desc)
}

// PaginatedResponse โครงสร้างการตอบกลับแบบแบ่งหน้า
type PaginationMeta struct {
	Page        int   `json:"page"`
	Limit       int   `json:"limit"`
	Total       int64 `json:"total"`
	TotalPages  int   `json:"totalPages"`
	HasNext     bool  `json:"hasNext"`
	HasPrevious bool  `json:"hasPrevious"`
}

type PaginatedResponse[T any] struct {
	Data []T            `json:"data"`
	Meta PaginationMeta `json:"meta"`
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
