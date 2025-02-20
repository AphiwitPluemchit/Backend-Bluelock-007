package models

type Order struct {
	ID    string `json:"id" bson:"_id,omitempty"`
	Name  string `json:"name" bson:"name"`
	Price string `json:"price" bson:"price"`
	Qty   string `json:"qty" bson:"qty"`
}
