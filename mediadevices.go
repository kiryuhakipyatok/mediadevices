package mediadevices

import (
	"fmt"

	"github.com/pion/mediadevices/pkg/driver"
)

var errNotFound = fmt.Errorf("failed to find the best driver that fits the constraints")

func EnumerateDevices() []MediaDeviceInfo {
	drivers := driver.GetManager().Query(
		driver.FilterFn(func(driver.Driver) bool { return true }))
	info := make([]MediaDeviceInfo, 0, len(drivers))
	for _, d := range drivers {
		var kind MediaDeviceType
		switch {
		case driver.FilterVideoRecorder()(d):
			kind = VideoInput
		default:
			continue
		}
		driverInfo := d.Info()
		var name string
		if driverInfo.Name == "" {
			name = "undefined name"
		} else {
			name = driverInfo.Name
		}
		info = append(info, MediaDeviceInfo{
			DeviceID:   d.ID(),
			Kind:       kind,
			Label:      driverInfo.Label,
			DeviceType: driverInfo.DeviceType,
			Name:       name,
		})
	}
	return info
}
