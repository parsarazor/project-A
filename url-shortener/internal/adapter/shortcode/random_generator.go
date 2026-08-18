package shortcode

import "github.com/url-shortener/internal/utils"

type RandomGenerator struct{}

func NewRandomGenerator() *RandomGenerator {
	return &RandomGenerator{}
}

func (g *RandomGenerator) Generate(length int) (string, error) {
	return utils.GenerateShortCode(length)
}
