package ports

type ShortCodeGenerator interface {
	Generate(length int) (string, error)
}
