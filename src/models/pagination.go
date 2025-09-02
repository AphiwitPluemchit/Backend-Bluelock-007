package models

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PaginationParams ใช้เก็บค่าการแบ่งหน้า, ค้นหา และเรียงลำดับ
type PaginationParams struct {
	Page   int     `json:"page" query:"page"  example:"1"`      // หมายเลขหน้าที่ต้องการ
	Limit  int     `json:"limit" query:"limit" example:"10"`    // จำนวนรายการต่อหน้า
	Search string  `json:"search" query:"search" example:""`    // คำค้นหา (Optional)
	SortBy string  `json:"sortBy" query:"sortBy" example:"_id"` // ฟิลด์ที่ใช้เรียงลำดับ
	Order  string  `json:"order" query:"order" example:"desc"`  // ทิศทางการเรียง (asc/desc)
	LastID *string `json:"lastId" query:"lastId"`
}

// PaginatedResponse โครงสร้างการตอบกลับแบบแบ่งหน้า
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// DefaultPagination ค่าตั้งต้นสำหรับ Pagination
func DefaultPagination() PaginationParams {
	return PaginationParams{
		Page:   1,
		Limit:  10,
		Search: "",
		SortBy: "_id",
		Order:  "desc",
	}
}

func CleanPagination(pagination PaginationParams) PaginationParams {
	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.Limit < 1 || pagination.Limit > 100 {
		pagination.Limit = 10
	}
	if pagination.SortBy == "" {
		pagination.SortBy = "_id"
	}
	if pagination.Order == "" {
		pagination.Order = "desc"
	}
	return pagination
}

// Paginate เป็นฟังก์ชันที่ใช้ทำ pagination ทั่วไป
func Paginate(ctx context.Context, collection *mongo.Collection, filter bson.M, sortField string, sortOrder string, page int, limit int, results interface{}) (PaginationMeta, error) {
	// คำนวณ skip
	skip := int64((page - 1) * limit)

	order := 1
	if sortOrder == "desc" {
		order = -1
	}

	// สร้าง Find Options
	findOptions := options.Find().SetSort(bson.D{{Key: sortField, Value: order}}).SetSkip(skip).SetLimit(int64(limit))

	// ดึงข้อมูล
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return PaginationMeta{}, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, results); err != nil {
		return PaginationMeta{}, err
	}

	// นับจำนวนเอกสารทั้งหมด (Total Count)
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return PaginationMeta{}, err
	}

	// คำนวณ TotalPages
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return PaginationMeta{
		Total:      total,
		Limit:      limit,
		Page:       page,
		TotalPages: totalPages,
	}, nil
}

// AggregatePaginate เป็นฟังก์ชันที่จัดการ pagination โดยใช้ Aggregation Pipeline
func AggregatePaginate(ctx context.Context, collection *mongo.Collection, pipeline mongo.Pipeline, page, limit int, results interface{}) (PaginationMeta, error) {

	facetStage := bson.D{{Key: "$facet", Value: bson.M{
		"metadata": []bson.M{
			{"$count": "total"},
			{"$addFields": bson.M{"page": page, "limit": limit}},
		},
		"data": []bson.M{
			{"$skip": (page - 1) * limit},
			{"$limit": limit},
		},
	}}}

	pipeline = append(pipeline, facetStage)

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return PaginationMeta{}, err
	}
	defer cursor.Close(ctx)

	var aggResults []struct {
		Metadata []struct {
			Total int64 `bson:"total"`
			Page  int   `bson:"page"`
			Limit int   `bson:"limit"`
		} `bson:"metadata"`
		Data []UploadCertificate `bson:"data"`
	}

	if err := cursor.All(ctx, &aggResults); err != nil {
		return PaginationMeta{}, err
	}

	if len(aggResults) == 0 || len(aggResults[0].Metadata) == 0 {
		return PaginationMeta{}, nil
	}

	meta := aggResults[0].Metadata[0]
	// Using a pointer to unmarshal into the provided slice
	*results.(*[]UploadCertificate) = aggResults[0].Data

	totalPages := int((meta.Total + int64(meta.Limit) - 1) / int64(meta.Limit))

	return PaginationMeta{
		Total:      meta.Total,
		Limit:      meta.Limit,
		Page:       meta.Page,
		TotalPages: totalPages,
	}, nil
}

// AggregatePaginateGlobal ทำงานกับ pipeline ใด ๆ ได้หมด
// - ไม่แก้ไข pipeline ต้นฉบับ (ทำสำเนาใหม่)
// - ใช้ $facet นับ total + ตัดหน้า (skip/limit) ในรอบเดียว
// - คืน []T และ meta (ไม่ต้องส่ง pointer slice เข้ามา)
// - ถ้าอยากกำหนด aggregate options เอง ส่งเข้ามาที่ opts (เช่น AllowDiskUse/MaxTime)
func AggregatePaginateGlobal[T any](
	ctx context.Context,
	coll *mongo.Collection,
	base mongo.Pipeline,
	page, limit int,
	opts ...*options.AggregateOptions,
) ([]T, PaginationMeta, error) {

	// safety defaults
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	// clone pipeline เพื่อไม่กระทบของเดิม
	pipeline := make(mongo.Pipeline, 0, len(base)+1)
	pipeline = append(pipeline, base...)

	// facet: metadata (count) + data (skip/limit)
	facet := bson.D{{Key: "$facet", Value: bson.M{
		"metadata": []bson.M{
			{"$count": "total"},
			{"$addFields": bson.M{"page": page, "limit": limit}},
		},
		"data": []bson.M{
			{"$skip": (page - 1) * limit},
			{"$limit": limit},
		},
	}}}
	pipeline = append(pipeline, facet)

	// aggregate options (ค่าเริ่มต้น AllowDiskUse = true)
	var aggOpt *options.AggregateOptions
	if len(opts) > 0 && opts[0] != nil {
		aggOpt = opts[0]
	} else {
		aggOpt = options.Aggregate().SetAllowDiskUse(true)
	}

	cur, err := coll.Aggregate(ctx, pipeline, aggOpt)
	if err != nil {
		return nil, PaginationMeta{}, err
	}
	defer cur.Close(ctx)

	// ซองสำหรับ decode ผลลัพธ์ (generic T)
	var envelope []struct {
		Metadata []struct {
			Total int64 `bson:"total"`
			Page  int   `bson:"page"`
			Limit int   `bson:"limit"`
		} `bson:"metadata"`
		Data []T `bson:"data"`
	}

	if err := cur.All(ctx, &envelope); err != nil {
		return nil, PaginationMeta{}, err
	}

	// ค่าเริ่มต้นเมื่อไม่มีผลลัพธ์เลย
	var (
		total int64 = 0
		pg          = page
		lm          = limit
		data  []T   = []T{}
	)

	if len(envelope) > 0 {
		// ตั้งค่า data แม้จะว่างก็ตาม
		data = envelope[0].Data
		if len(envelope[0].Metadata) > 0 {
			total = envelope[0].Metadata[0].Total
			pg = envelope[0].Metadata[0].Page
			lm = envelope[0].Metadata[0].Limit
		}
	}

	totalPages := 0
	if lm > 0 {
		totalPages = int((total + int64(lm) - 1) / int64(lm))
	}

	meta := PaginationMeta{
		Page:       pg,
		Limit:      lm,
		Total:      total,
		TotalPages: totalPages,
	}
	return data, meta, nil
}

// ---------- options pattern ----------
type aggPaginateConfig struct {
	AllowDiskUse   bool
	StableSortByID bool // เติม $sort {_id: 1} ถ้า pipeline ยังไม่มี $sort
}

type AggPaginateOption func(*aggPaginateConfig)

func WithAllowDiskUse() AggPaginateOption {
	return func(c *aggPaginateConfig) { c.AllowDiskUse = true }
}

func WithStableSortByID() AggPaginateOption {
	return func(c *aggPaginateConfig) { c.StableSortByID = true }
}

// ---------- helper ----------
func hasSortStage(p mongo.Pipeline) bool {
	for _, st := range p {
		if len(st) > 0 && st[0].Key == "$sort" {
			return true
		}
	}
	return false
}

// AggregatePaginate[T] รัน aggregation + แปลงผลเป็น []T
// - ใส่ $facet { metadata: [$count], data: [$skip, $limit] } ให้อัตโนมัติ
// - total = 0 → คืน slice ว่าง และ meta ที่ page/limit ตรงตามอินพุต
// - ใช้ได้กับทุก collection/โครงสร้าง: เรียกด้วย T ที่ต้องการ เช่น T = models.UploadCertificate, models.Student, หรือ bson.M
func AggregatePaginate2[T any](
	ctx context.Context,
	coll *mongo.Collection,
	basePipeline mongo.Pipeline,
	page, limit int,
	optFns ...AggPaginateOption,
) ([]T, PaginationMeta, error) {

	// sanitize page/limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	// apply options
	cfg := &aggPaginateConfig{}
	for _, f := range optFns {
		f(cfg)
	}

	pipeline := make(mongo.Pipeline, 0, len(basePipeline)+2)
	pipeline = append(pipeline, basePipeline...)

	// เสริมความเสถียรของลำดับผลลัพธ์ (ป้องกันโดดหน้า/ซ้ำหน้า) ถ้า caller ไม่ได้ใส่ $sort มา
	if cfg.StableSortByID && !hasSortStage(pipeline) {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}})
	}

	// facet นับทั้งหมดก่อน แล้วค่อยตัดหน้าใน data
	facetStage := bson.D{{Key: "$facet", Value: bson.M{
		"metadata": []bson.D{
			{{Key: "$count", Value: "total"}},
			{{Key: "$addFields", Value: bson.M{"page": page, "limit": limit}}},
		},
		"data": []bson.D{
			{{Key: "$skip", Value: (page - 1) * limit}},
			{{Key: "$limit", Value: limit}},
		},
	}}}
	pipeline = append(pipeline, facetStage)

	aggOpts := options.Aggregate()
	if cfg.AllowDiskUse {
		aggOpts.SetAllowDiskUse(true)
	}

	cur, err := coll.Aggregate(ctx, pipeline, aggOpts)
	if err != nil {
		return nil, PaginationMeta{}, err
	}
	defer cur.Close(ctx)

	// โครงสร้างรับผลจาก $facet → แปลง data เป็น []T
	var out []struct {
		Metadata []struct {
			Total int64 `bson:"total"`
			Page  int   `bson:"page"`
			Limit int   `bson:"limit"`
		} `bson:"metadata"`
		Data []T `bson:"data"`
	}

	if err := cur.All(ctx, &out); err != nil {
		return nil, PaginationMeta{}, err
	}

	// ค่าเริ่มต้น (กรณีไม่มีเอกสารเลย)
	total := int64(0)
	pg := page
	lm := limit
	data := make([]T, 0)

	if len(out) > 0 {
		if len(out[0].Metadata) > 0 {
			total = out[0].Metadata[0].Total
			pg = out[0].Metadata[0].Page
			lm = out[0].Metadata[0].Limit
		}
		// ต่อให้ metadata ว่าง ก็ยัง set data เป็น slice ว่างแทน nil
		data = out[0].Data
	}

	totalPages := 0
	if lm > 0 {
		totalPages = int((total + int64(lm) - 1) / int64(lm))
	}

	return data, PaginationMeta{
		Page:       pg,
		Limit:      lm,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}
