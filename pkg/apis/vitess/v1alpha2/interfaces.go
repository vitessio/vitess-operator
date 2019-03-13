package v1alpha2

type ConfigProvider interface {
	GetTabletContainers() *TabletContainers
}
