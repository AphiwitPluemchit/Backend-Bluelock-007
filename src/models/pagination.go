package models

// PaginationParams ใช้เก็บค่าการแบ่งหน้า, ค้นหา และเรียงลำดับ
type PaginationParams struct {
	Page   int    `json:"page" query:"page"`     // หมายเลขหน้าที่ต้องการ
	Limit  int    `json:"limit" query:"limit"`   // จำนวนรายการต่อหน้า
	Search string `json:"search" query:"search"` // คำค้นหา (Optional)
	SortBy string `json:"sortBy" query:"sortBy"` // ฟิลด์ที่ใช้เรียงลำดับ
	Order  string `json:"order" query:"order"`   // ทิศทางการเรียง (asc/desc)
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
