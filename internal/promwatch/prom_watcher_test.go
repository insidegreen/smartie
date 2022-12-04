package promwatch

import (
	"testing"
)

func TestWatch(t *testing.T) {

	promWatch := New("http://192.168.178.22:9090")

	promWatch.PromQuery(`node_power_supply_current_capacity * on(instance) group_left(nodename) node_uname_info >= 100`)

}
