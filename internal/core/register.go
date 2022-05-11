package core

type creator func(pp *ProxyPool) IGetter

var creatorMap = make(map[string]creator)

func Register(sourceType string, c creator) {
	creatorMap[sourceType] = c
}

func GetAllGetters(pp *ProxyPool) []IGetter {
	getters := make([]IGetter, 0, len(creatorMap))
	for _, creatorFnc := range creatorMap {
		getters = append(getters, creatorFnc(pp))
	}
	return getters
}
