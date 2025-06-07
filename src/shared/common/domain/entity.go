package domain

// entity ต้องมี id
type Entity[TId any] struct {
	ID TId
}
