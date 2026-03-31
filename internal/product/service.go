package product

type Service struct {
	products []Product
}

func NewService() *Service {
	return &Service{
		products: []Product{
			{ID: 1, Name: "Sample Product", Description: "Static product record", PriceCents: 1999},
			{ID: 2, Name: "Second Product", Description: "Another static record", PriceCents: 2999},
			{ID: 3, Name: "Third Product", Description: "Third static record", PriceCents: 3999},
		},
	}
}

func (s *Service) List() []Product {
	products := make([]Product, len(s.products))
	copy(products, s.products)
	return products
}
