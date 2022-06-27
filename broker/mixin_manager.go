package broker

type MixinManager interface {
	GetForwards(tag string) []string
}
