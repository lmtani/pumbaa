package ports

type Prompt interface {
	SelectByKey(taskOptions []string) (string, error)
	SelectByIndex(sfn func(input string, index int) bool, items interface{}) (int, error)
}
