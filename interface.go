package roomapi

type Browser interface {
	Scrape(int) string
	Retrieve()
}

type Product interface {
	Get()
	Save()
	Delete()
	Update()
	Browser
}

type User Product

type Website interface {
	Product
}
