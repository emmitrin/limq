package channel

type MixinManager interface {
	GetForwards(tag string) []string
}
