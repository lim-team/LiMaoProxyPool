package core

type IGetter interface {
	Run(ipChan chan *IP)
}
