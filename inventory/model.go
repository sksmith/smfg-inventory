package inventory

// Product is a value object. A SKU able to be produced by the factory.
type Product struct {
	Sku string
	Upc string
	Name string
}

// Unit is an entity. A single physical object in the warehouse.
type Unit struct {
	Type Product
	LocationID uint64
	ContainerID uint64
}

// Location is an entity. A physical space in a physical warehouse or factory.
type Location struct {
	ID uint64
	FacilityID uint64
	Description string
}

// Facility is an entity. All or part of a physical building.
type Facility struct {
	ID uint64
	Name string
}

// Container is an entity. A physical pallet, gaylord, or carton containing units.
type Container struct {
	ID uint64
	LocationID uint64
}

// Inventory is a value object. A current level of inventory for each product in a Facility.
type Inventory struct {
	LocationID uint64
	Sku string
	Available int64
	Reserved int64
}

// Reservation is an entity. An amount of inventory set aside for a given Customer.
type Reservation struct {
	ID uint64
	Sku string
	RequestedAmount int64
	ReservedAmount int64
}